package handlers

import (
	"encoding/json"
	"errors"
	"sort"

	"golang.org/x/exp/slices"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

type SoftwareInterface interface {
	GetAllSoftware(ctx *fiber.Ctx) error
	GetSoftware(ctx *fiber.Ctx) error
	PostSoftware(ctx *fiber.Ctx) error
	PatchSoftware(ctx *fiber.Ctx) error
	DeleteSoftware(ctx *fiber.Ctx) error
}

type Software struct {
	db *gorm.DB
}

var (
	errLoadNotFound       = errors.New("Software was not found")
	errLoad               = errors.New("error while loading Software")
	errMalformedJSONPatch = errors.New("malformed JSON Patch")
)

func NewSoftware(db *gorm.DB) *Software {
	return &Software{db: db}
}

// GetAllSoftware gets the list of all software and returns any error encountered.
func (p *Software) GetAllSoftware(ctx *fiber.Ctx) error { //nolint:cyclop // mostly error handling ifs
	var software []models.Software

	// Preload will load all the associated aliases, which include
	// also the canonical url. We'll manually handle that later.
	stmt := p.db.Preload("Aliases")

	stmt, err := general.Clauses(ctx, stmt, "")
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Software",
			err.Error(),
		)
	}

	// Return just software with a certain URL if the 'url' query filter
	// is used.
	if url := ctx.Query("url", ""); url != "" {
		var softwareURL models.SoftwareURL

		err = p.db.First(&softwareURL, "url = ?", url).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(
				fiber.StatusInternalServerError,
				"can't get Software",
				fiber.ErrInternalServerError.Message,
			)
		}

		stmt.Where("id = ?", softwareURL.SoftwareID)
	} else {
		if all := ctx.QueryBool("all", false); !all {
			stmt = stmt.Scopes(models.Active)
		}
	}

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &software)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Software",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
	}

	// Remove the canonical URL from the aliases, because it need to be its own
	// field. It was loaded previously together with the other aliases in Preload(),
	// because of limitation in gorm.
	for swIdx := range software {
		swr := &software[swIdx]

		for aliasIdx := range swr.Aliases {
			alias := &swr.Aliases[aliasIdx]

			if alias.ID == swr.SoftwareURLID {
				swr.URL = *alias

				swr.Aliases[aliasIdx] = swr.Aliases[len(swr.Aliases)-1]
				swr.Aliases = swr.Aliases[:len(swr.Aliases)-1]

				break
			}
		}
	}

	return ctx.JSON(fiber.Map{"data": &software, "links": general.PaginationLinks(cursor)})
}

// GetSoftware gets the software with the given ID and returns any error encountered.
func (p *Software) GetSoftware(ctx *fiber.Ctx) error {
	const errMsg = "can't get Software"

	software := models.Software{}

	if err := loadSoftware(p.db, &software, ctx.Params("id")); err != nil {
		if errors.Is(err, errLoadNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
		}

		return common.InternalServerError(errMsg)
	}

	return ctx.JSON(&software)
}

// PostSoftware creates a new software.
func (p *Software) PostSoftware(ctx *fiber.Ctx) error {
	const errMsg = "can't create Software"

	softwareReq := new(common.SoftwarePost)

	if err := common.ValidateRequestEntity(ctx, softwareReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	aliases := []models.SoftwareURL{}
	for _, u := range softwareReq.Aliases {
		aliases = append(aliases, models.SoftwareURL{ID: utils.UUIDv4(), URL: u})
	}

	url := models.SoftwareURL{ID: utils.UUIDv4(), URL: softwareReq.URL}
	software := models.Software{
		ID: utils.UUIDv4(),

		// Manually set the URL and its foreign key because of a limitation in gorm
		URL:           url,
		SoftwareURLID: url.ID,

		Aliases:       aliases,
		PubliccodeYml: softwareReq.PubliccodeYml,
		Active:        softwareReq.Active,
		Vitality:      softwareReq.Vitality,
	}

	if err := p.db.Create(&software).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	return ctx.JSON(&software)
}

// PatchSoftware updates the software with the given ID.
func (p *Software) PatchSoftware(ctx *fiber.Ctx) error { //nolint:funlen,cyclop
	const errMsg = "can't update Software"

	software := models.Software{}

	if err := loadSoftware(p.db, &software, ctx.Params("id")); err != nil {
		if errors.Is(err, errLoadNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	softwareJSON, err := json.Marshal(&software)
	if err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	var updatedJSON []byte

	switch ctx.Get(fiber.HeaderContentType) {
	case "application/json-patch+json":
		patch, err := jsonpatch.DecodePatch(ctx.Body())
		if err != nil {
			return common.Error(fiber.StatusBadRequest, errMsg, errMalformedJSONPatch.Error())
		}

		updatedJSON, err = patch.Apply(softwareJSON)
		if err != nil {
			return common.Error(fiber.StatusUnprocessableEntity, errMsg, err.Error())
		}

	// application/merge-patch+json by default
	default:
		softwareReq := common.SoftwarePatch{}
		if err := common.ValidateRequestEntity(ctx, &softwareReq, errMsg); err != nil {
			return err //nolint:wrapcheck
		}

		updatedJSON, err = jsonpatch.MergePatch(softwareJSON, ctx.Body())
		if err != nil {
			return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
		}
	}

	var updatedSoftware models.Software

	err = json.Unmarshal(updatedJSON, &updatedSoftware)
	if err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	// Slice of aliases that we expect to be in the database after the PATCH
	expectedAliases := make([]string, 0, len(updatedSoftware.Aliases))
	for _, alias := range updatedSoftware.Aliases {
		expectedAliases = append(expectedAliases, alias.URL)
	}

	if err := p.db.Transaction(func(tran *gorm.DB) error {
		//nolint:gocritic // it's fine, we want to append to another slice
		currentURLs := append(software.Aliases, software.URL)

		updatedURL, aliases, err := syncAliases(
			tran,
			software.ID,
			currentURLs,
			updatedSoftware.URL.URL,
			expectedAliases,
		)
		if err != nil {
			return err
		}

		// Manually set the canonical URL via the foreign key because of a limitation in gorm
		updatedSoftware.SoftwareURLID = updatedURL.ID
		updatedSoftware.URL = *updatedURL

		// Set Aliases to a zero value, so it's not touched by gorm's Update(),
		// because we handle the alias manually
		updatedSoftware.Aliases = []models.SoftwareURL{}

		if err := tran.Updates(&updatedSoftware).Error; err != nil {
			return err
		}

		updatedSoftware.Aliases = aliases

		return nil
	}); err != nil {
		switch {
		case errors.Is(err, gorm.ErrDuplicatedKey):
			return common.Error(fiber.StatusConflict, errMsg, "URL already exists")
		default:
			//nolint:wrapcheck // default to not wrap other errors, the handler will take care of this
			return err
		}
	}

	// Sort the aliases to always have a consistent output
	sort.Slice(updatedSoftware.Aliases, func(a int, b int) bool {
		return updatedSoftware.Aliases[a].URL < updatedSoftware.Aliases[b].URL
	})

	return ctx.JSON(&updatedSoftware)
}

// DeleteSoftware deletes the software with the given ID.
func (p *Software) DeleteSoftware(ctx *fiber.Ctx) error {
	result := p.db.Select("Aliases").Delete(&models.Software{ID: ctx.Params("id")})

	if result.Error != nil {
		return common.Error(fiber.StatusInternalServerError, "can't delete Software", "db error")
	}

	if result.RowsAffected == 0 {
		return common.Error(fiber.StatusNotFound, "can't delete Software", "Software was not found")
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func loadSoftware(gormdb *gorm.DB, software *models.Software, id string) error {
	if err := gormdb.First(&software, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errLoadNotFound
		}

		return errLoad
	}

	if err := gormdb.
		Where("software_id = ? AND id <> ?", software.ID, software.SoftwareURLID).Find(&software.Aliases).
		Error; err != nil {
		return errLoad
	}

	if err := gormdb.Debug().Where("id = ?", software.SoftwareURLID).First(&software.URL).Error; err != nil {
		return errLoad
	}

	return nil
}

// syncAliases synchs the SoftwareURLs for a `Software` in the database to reflect the
// passed list of `expectedAliases` and the canonical `url`.
//
// It returns the new canonical SoftwareURL and the new slice of aliases or an error if any.
func syncAliases( //nolint:cyclop // mostly error handling ifs
	gormdb *gorm.DB,
	softwareID string,
	currentURLs []models.SoftwareURL,
	expectedURL string,
	expectedAliases []string,
) (*models.SoftwareURL, []models.SoftwareURL, error) {
	toRemove := []string{}          // Slice of SoftwareURL ids to remove from the database
	toAdd := []models.SoftwareURL{} // Slice of SoftwareURLs to add to the database

	// Map mirroring the state of SoftwareURLs for this software in the database,
	// keyed by url
	urlMap := map[string]models.SoftwareURL{}

	for _, url := range currentURLs {
		urlMap[url.URL] = url
	}

	//nolint:gocritic // it's fine, we want to another slice
	allSoftwareURLs := append(expectedAliases, expectedURL)

	for urlStr, softwareURL := range urlMap {
		if !slices.Contains(allSoftwareURLs, urlStr) {
			toRemove = append(toRemove, softwareURL.ID)

			delete(urlMap, urlStr)
		}
	}

	for _, urlStr := range allSoftwareURLs {
		_, exists := urlMap[urlStr]
		if !exists {
			su := models.SoftwareURL{ID: utils.UUIDv4(), URL: urlStr, SoftwareID: softwareID}

			toAdd = append(toAdd, su)
			urlMap[urlStr] = su
		}
	}

	if len(toRemove) > 0 {
		if err := gormdb.Delete(&models.SoftwareURL{}, toRemove).Error; err != nil {
			return nil, nil, err
		}
	}

	if len(toAdd) > 0 {
		if err := gormdb.Create(toAdd).Error; err != nil {
			return nil, nil, err
		}
	}

	updatedURL := urlMap[expectedURL]

	// Remove the canonical URL from the rest of the URLs, so we can return
	// URL and aliases in different fields.
	delete(urlMap, expectedURL)

	aliases := make([]models.SoftwareURL, 0, len(urlMap))
	for _, alias := range urlMap {
		aliases = append(aliases, alias)
	}

	return &updatedURL, aliases, nil
}
