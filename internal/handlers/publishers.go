package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"sort"

	"golang.org/x/exp/slices"

	"github.com/italia/developers-italia-api/internal/handlers/general"

	"github.com/gofiber/fiber/v2/utils"

	"github.com/PuerkitoBio/purell"
	"github.com/gofiber/fiber/v2"
	"github.com/italia/developers-italia-api/internal/common"
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

const normalizeFlags = purell.FlagsUsuallySafeGreedy | purell.FlagRemoveWWW

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

	if all := ctx.QueryBool("all", false); !all {
		stmt = stmt.Scopes(models.Active)
	}

	paginator := general.NewPaginator(ctx)

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

		return common.Error(fiber.StatusInternalServerError, "can't get Publisher", "internal server error")
	}

	return ctx.JSON(&publisher)
}

// PostPublisher creates a new publisher.
func (p *Publisher) PostPublisher(ctx *fiber.Ctx) error {
	request := new(common.PublisherPost)

	err := common.ValidateRequestEntity(ctx, request, "can't create Publisher")
	if err != nil {
		return err //nolint:wrapcheck
	}

	if request.AlternativeID != nil {
		//nolint:godox // postpone the fix
		// FIXME: Possible TOCTTOU race here
		result := p.db.Limit(1).Find(&models.Publisher{ID: *request.AlternativeID})

		if result.Error != nil {
			return common.Error(fiber.StatusInternalServerError, "can't create Publisher", "db error")
		}

		if result.RowsAffected != 0 {
			return common.Error(fiber.StatusConflict,
				"can't create Publisher",
				fmt.Sprintf("Publisher with id '%s' already exists", *request.AlternativeID),
			)
		}
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
		urlAddress, _ := url.Parse(codeHost.URL)
		normalizedURL := purell.NormalizeURL(urlAddress, normalizeFlags)

		publisher.CodeHosting = append(publisher.CodeHosting,
			models.CodeHosting{
				ID:    utils.UUIDv4(),
				URL:   normalizedURL,
				Group: codeHost.Group,
			})
	}

	if err := p.db.Create(&publisher).Error; err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return common.Error(fiber.StatusNotFound,
				"can't create Publisher",
				"Publisher was not found")
		case errors.Is(err, gorm.ErrDuplicatedKey):
			return common.Error(fiber.StatusConflict,
				"can't create Publisher",
				"description, alternativeId or codeHosting URL already exists")
		default:
			return common.Error(fiber.StatusInternalServerError,
				"can't create Publisher",
				"internal server error")
		}
	}

	return ctx.JSON(&publisher)
}

// PatchPublisher updates the publisher with the given ID. CodeHosting URLs will be overwritten from the request.
func (p *Publisher) PatchPublisher(ctx *fiber.Ctx) error { //nolint:cyclop,funlen // mostly error handling ifs
	publisherReq := new(common.PublisherPatch)
	publisher := models.Publisher{}
	id := ctx.Params("id")

	// Preload will load all the associated CodeHosting. We'll manually handle that later.
	if err := p.db.Preload("CodeHosting").First(&publisher, "id = ? or alternative_id = ?", id, id).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't update Publisher", "Publisher was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't update Publisher",
			fiber.ErrInternalServerError.Message,
		)
	}

	if err := common.ValidateRequestEntity(ctx, publisherReq, "can't update Publisher"); err != nil {
		return err //nolint:wrapcheck
	}

	if publisherReq.AlternativeID != nil {
		//nolint:godox // postpone the fix
		// FIXME: Possible TOCTTOU race here
		result := p.db.Limit(1).Find(&models.Publisher{ID: *publisherReq.AlternativeID})

		if result.Error != nil {
			return common.Error(fiber.StatusInternalServerError, "can't update Publisher", "db error")
		}

		if result.RowsAffected != 0 {
			return common.Error(fiber.StatusConflict,
				"can't update Publisher",
				fmt.Sprintf("Publisher with id '%s' already exists", *publisherReq.AlternativeID),
			)
		}
	}

	// Slice of CodeHosting URLs that we expect in the database after the PATCH
	var expectedURLs []string

	// application/merge-patch+json semantics: change CodeHosting only if
	// the request specifies a "CodeHosting" key.
	if publisherReq.CodeHosting != nil {
		for _, ch := range *publisherReq.CodeHosting {
			expectedURLs = append(expectedURLs, purell.MustNormalizeURLString(ch.URL, normalizeFlags))
		}
	} else {
		for _, ch := range publisher.CodeHosting {
			expectedURLs = append(expectedURLs, ch.URL)
		}
	}

	if err := p.db.Transaction(func(tran *gorm.DB) error {
		codeHosting, err := syncCodeHosting(tran, publisher, expectedURLs)
		if err != nil {
			return err
		}

		if publisherReq.Description != nil {
			publisher.Description = *publisherReq.Description
		}
		if publisherReq.Email != nil {
			publisher.Email = common.NormalizeEmail(publisherReq.Email)
		}
		if publisherReq.Active != nil {
			publisher.Active = publisherReq.Active
		}
		if publisher.AlternativeID != nil {
			publisher.AlternativeID = publisherReq.AlternativeID
		}

		// Set CodeHosting to a zero value, so it's not touched by gorm's Update(),
		// because we handle the alias manually
		publisher.CodeHosting = []models.CodeHosting{}

		if err := tran.Updates(&publisher).Error; err != nil {
			return err
		}

		publisher.CodeHosting = codeHosting

		return nil
	}); err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't update Publisher", err.Error())
	}

	// Sort the aliases to always have a consistent output
	sort.Slice(publisher.CodeHosting, func(a int, b int) bool {
		return publisher.CodeHosting[a].URL < publisher.CodeHosting[b].URL
	})

	return ctx.JSON(&publisher)
}

// DeletePublisher deletes the publisher with the given ID.
func (p *Publisher) DeletePublisher(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	result := p.db.Select("CodeHosting").Where("id = ? or alternative_id = ?", id, id).Delete(&models.Publisher{})

	if result.Error != nil {
		return common.Error(fiber.StatusInternalServerError, "can't delete Publisher", "db error")
	}

	if result.RowsAffected == 0 {
		return common.Error(fiber.StatusNotFound, "can't delete Publisher", "Publisher was not found")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
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
