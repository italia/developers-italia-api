package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"
)

type PublisherInterface interface {
	GetPublishers(ctx *fiber.Ctx) error
	GetPublisher(ctx *fiber.Ctx) error
	PostPublisher(ctx *fiber.Ctx) error
	PatchPublisher(ctx *fiber.Ctx) error
	DeletePublisher(ctx *fiber.Ctx) error
}

const (
	alreadyExists        = "already exists"
	contentTypeJSONPatch = "application/json-patch+json"
)

type Publisher struct {
	db *gorm.DB
}

func NewPublisher(db *gorm.DB) *Publisher {
	return &Publisher{db: db}
}

// GetPublishers gets the list of all publishers and returns any error encountered.
func (p *Publisher) GetPublishers(ctx *fiber.Ctx) error {
	var publishers []models.Publisher

	stmt := p.db.Preload("CodeHosting")

	stmt, err := general.Clauses(ctx, stmt, "")
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Publishers",
			err.Error(),
		)
	}

	if all := ctx.QueryBool("all", false); !all {
		stmt = stmt.Scopes(models.Active)
	}

	paginator, err := general.NewPaginator(ctx)
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Publishers", err.Error())
	}

	result, cursor, err := paginator.Paginate(stmt, &publishers)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Publishers",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Publisher",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &publishers, "links": general.PaginationLinks(cursor)})
}

// GetPublisher gets the publisher with the given ID and returns any error encountered.
func (p *Publisher) GetPublisher(ctx *fiber.Ctx) error {
	publisher := models.Publisher{}
	id := ctx.Params("id")

	if err := p.db.Preload("CodeHosting").First(&publisher, "id = ? or alternative_id = ?", id, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Publisher", "Publisher was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Publisher",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(&publisher)
}

// PostPublisher creates a new publisher.
func (p *Publisher) PostPublisher(ctx *fiber.Ctx) error {
	const errMsg = "can't create Publisher"

	request := new(common.PublisherPost)

	if err := common.ValidateRequestEntity(ctx, request, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	normalizedEmail := common.NormalizeEmail(request.Email)

	publisher := &models.Publisher{
		ID:            utils.UUIDv4(),
		Description:   request.Description,
		Email:         normalizedEmail,
		Active:        request.Active,
		AlternativeID: request.AlternativeID,
	}

	for _, codeHost := range request.CodeHosting {
		publisher.CodeHosting = append(publisher.CodeHosting,
			models.CodeHosting{
				ID:    utils.UUIDv4(),
				URL:   common.NormalizeURL(codeHost.URL),
				Group: codeHost.Group,
			})
	}

	if err := p.db.Transaction(func(tran *gorm.DB) error {
		if request.AlternativeID != nil {
			if err := checkAlternativeIDConflict(tran, *request.AlternativeID); err != nil {
				return err
			}
		}

		return tran.Create(&publisher).Error
	}); err != nil {
		var idConflict idConflictError

		if errors.As(err, &idConflict) {
			return common.Error(fiber.StatusConflict, errMsg, idConflict.Error())
		}

		if field := common.DuplicateField(err); field != nil {
			detail := alreadyExists
			if *field != "" {
				detail = *field + " " + alreadyExists
			}

			return common.Error(fiber.StatusConflict, errMsg, detail)
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID.
// Supports both JSON Merge Patch (default) and JSON Patch (application/json-patch+json).
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error { //nolint:cyclop,funlen,gocognit // mostly error handling ifs
	const errMsg = "can't update Publisher"

	publisher := models.Publisher{}
	id := ctx.Params("id")

	// Preload will load all the associated CodeHosting. We'll manually handle that later.
	if err := p.db.Preload("CodeHosting").First(&publisher, "id = ? or alternative_id = ?", id, id).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	publisherJSON, err := json.Marshal(&publisher)
	if err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	var updatedJSON []byte

	switch ctx.Get(fiber.HeaderContentType) {
	case contentTypeJSONPatch:
		patch, err := jsonpatch.DecodePatch(ctx.Body())
		if err != nil {
			return common.Error(fiber.StatusBadRequest, errMsg, errMalformedJSONPatch.Error())
		}

		updatedJSON, err = patch.Apply(publisherJSON)
		if err != nil {
			return common.Error(fiber.StatusUnprocessableEntity, errMsg, err.Error())
		}

	// application/merge-patch+json by default
	default:
		publisherReq := new(common.PublisherPatch)

		if err := common.ValidateRequestEntity(ctx, publisherReq, errMsg); err != nil {
			return err //nolint:wrapcheck
		}

		updatedJSON, err = jsonpatch.MergePatch(publisherJSON, ctx.Body())
		if err != nil {
			return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
		}
	}

	var updatedPublisher models.Publisher

	if err := json.Unmarshal(updatedJSON, &updatedPublisher); err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	// Prevent patches from changing the ID.
	updatedPublisher.ID = publisher.ID

	updatedPublisher.Email = common.NormalizeEmail(updatedPublisher.Email)

	expectedURLs := make([]string, 0, len(updatedPublisher.CodeHosting))

	for _, ch := range updatedPublisher.CodeHosting {
		expectedURLs = append(expectedURLs, common.NormalizeURL(ch.URL))
	}

	if err := p.db.Transaction(func(tran *gorm.DB) error { //nolint:dupl
		if updatedPublisher.AlternativeID != nil &&
			(publisher.AlternativeID == nil || *updatedPublisher.AlternativeID != *publisher.AlternativeID) {
			if err := checkAlternativeIDConflict(tran, *updatedPublisher.AlternativeID); err != nil {
				return err
			}
		}

		codeHosting, err := syncCodeHosting(tran, publisher, expectedURLs)
		if err != nil {
			return err
		}

		publisher.Description = updatedPublisher.Description
		publisher.Email = updatedPublisher.Email
		publisher.Active = updatedPublisher.Active
		publisher.AlternativeID = updatedPublisher.AlternativeID

		// Set CodeHosting to a zero value, so it's not touched by gorm's Update(),
		// because we handle it manually via syncCodeHosting.
		publisher.CodeHosting = []models.CodeHosting{}

		if err := tran.Updates(&publisher).Error; err != nil {
			return err
		}

		publisher.CodeHosting = codeHosting

		return nil
	}); err != nil {
		var idConflict idConflictError

		if errors.As(err, &idConflict) {
			return common.Error(fiber.StatusConflict, errMsg, idConflict.Error())
		}

		if field := common.DuplicateField(err); field != nil {
			detail := alreadyExists
			if *field != "" {
				detail = *field + " " + alreadyExists
			}

			return common.Error(fiber.StatusConflict, errMsg, detail)
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	// Sort codeHosting to always have a consistent output.
	sort.Slice(publisher.CodeHosting, func(a int, b int) bool {
		return publisher.CodeHosting[a].URL < publisher.CodeHosting[b].URL
	})

	return ctx.JSON(&publisher)
}

// DeletePublisher deletes the publisher with the given ID.
func (p *Publisher) DeletePublisher(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	publisher := models.Publisher{}
	if err := p.db.First(&publisher, "id = ? or alternative_id = ?", id, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't delete Publisher", "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't delete Publisher", "db error")
	}

	result := p.db.Select("CodeHosting").Delete(&publisher)

	if result.Error != nil {
		return common.Error(fiber.StatusInternalServerError, "can't delete Publisher", "db error")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// idConflictError is returned when alternativeId conflicts with an existing publisher's primary key.
type idConflictError string

func (e idConflictError) Error() string {
	return fmt.Sprintf("Publisher with id '%s' already exists", string(e))
}

// checkAlternativeIDConflict returns idConflictError if any publisher exists whose primary key
// equals the given alternativeID value, which would cause ambiguous lookups.
func checkAlternativeIDConflict(db *gorm.DB, alternativeID string) error {
	result := db.Limit(1).Find(&models.Publisher{ID: alternativeID})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 0 {
		return idConflictError(alternativeID)
	}

	return nil
}

// syncCodeHosting synchs the CodeHosting for a `publisher` in the database to reflect the
// passed slice of `codeHosting` URLs.
//
// It returns the slice of CodeHosting in the database.
func syncCodeHosting( //nolint:cyclop // mostly error handling ifs
	gormdb *gorm.DB, publisher models.Publisher, codeHosting []string,
) ([]models.CodeHosting, error) {
	toRemove := []string{}          // Slice of CodeHosting ids to remove from the database
	toAdd := []models.CodeHosting{} // Slice of CodeHosting to add to the database

	// Map mirroring the state of CodeHosting for this software in the database,
	// keyed by url
	urlMap := map[string]models.CodeHosting{}

	for _, ch := range publisher.CodeHosting {
		urlMap[ch.URL] = ch
	}

	for url, ch := range urlMap {
		if !slices.Contains(codeHosting, url) {
			toRemove = append(toRemove, ch.ID)

			delete(urlMap, url)
		}
	}

	for _, url := range codeHosting {
		_, exists := urlMap[url]
		if !exists {
			ch := models.CodeHosting{ID: utils.UUIDv4(), URL: url, PublisherID: publisher.ID}

			toAdd = append(toAdd, ch)
			urlMap[url] = ch
		}
	}

	if len(toRemove) > 0 {
		if err := gormdb.Delete(&models.CodeHosting{}, toRemove).Error; err != nil {
			return nil, err
		}
	}

	if len(toAdd) > 0 {
		if err := gormdb.Create(toAdd).Error; err != nil {
			return nil, err
		}
	}

	retCodeHosting := make([]models.CodeHosting, 0, len(urlMap))
	for _, ch := range urlMap {
		retCodeHosting = append(retCodeHosting, ch)
	}

	return retCodeHosting, nil
}
