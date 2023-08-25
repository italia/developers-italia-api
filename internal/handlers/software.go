package handlers

import (
	"errors"
	"sort"

	"golang.org/x/exp/slices"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"
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
	software := models.Software{}

	if err := p.db.First(&software, "id = ?", ctx.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Software", "Software was not found")
		}

		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
	}

	if err := p.db.
		Where("software_id = ? AND id <> ?", software.ID, software.SoftwareURLID).Find(&software.Aliases).
		Error; err != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
	}

	if err := p.db.Where("id = ?", software.SoftwareURLID).First(&software.URL).Error; err != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Software",
			fiber.ErrInternalServerError.Message,
		)
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
	}

	if err := p.db.Create(&software).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	return ctx.JSON(&software)
}

// PatchSoftware updates the software with the given ID.
func (p *Software) PatchSoftware(ctx *fiber.Ctx) error {
	const errMsg = "can't update Software"

	softwareReq := new(common.SoftwarePatch)
	software := models.Software{}

	// Preload will load all the associated aliases, which include
	// also the canonical url. We'll manually handle that later.
	if err := p.db.Preload("Aliases").First(&software, "id = ?", ctx.Params("id")).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't update Software", "Software was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't update Software", "internal server error")
	}

	if err := common.ValidateRequestEntity(ctx, softwareReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	// Slice of urls that we expect in the database after the PATCH (url + aliases)
	var expectedURLs []string

	// application/merge-patch+json semantics: change url only if
	// the request specifies an "url" key.
	url := software.URL.URL
	if softwareReq.URL != "" {
		url = softwareReq.URL
	}

	// application/merge-patch+json semantics: change aliases only if
	// the request specifies an "aliases" key.
	if softwareReq.Aliases != nil {
		expectedURLs = *softwareReq.Aliases
	} else {
		for _, alias := range software.Aliases {
			expectedURLs = append(expectedURLs, alias.URL)
		}
	}

	expectedURLs = append(expectedURLs, url)

	if err := p.db.Transaction(func(tran *gorm.DB) error {
		updatedURL, aliases, err := syncAliases(tran, software, url, expectedURLs)
		if err != nil {
			return err
		}

		software.PubliccodeYml = softwareReq.PubliccodeYml
		software.Active = softwareReq.Active

		// Manually set the canonical URL via the foreign key because of a limitation in gorm
		software.SoftwareURLID = updatedURL.ID
		software.URL = *updatedURL

		// Set Aliases to a zero value, so it's not touched by gorm's Update(),
		// because we handle the alias manually
		software.Aliases = []models.SoftwareURL{}

		if err := tran.Updates(&software).Error; err != nil {
			return err
		}

		software.Aliases = aliases

		return nil
	}); err != nil {
		return common.Error(fiber.StatusInternalServerError, "can't update Software", err.Error())
	}

	// Sort the aliases to always have a consistent output
	sort.Slice(software.Aliases, func(a int, b int) bool {
		return software.Aliases[a].URL < software.Aliases[b].URL
	})

	return ctx.JSON(&software)
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

// syncAliases synchs the SoftwareURLs for a `software` in the database to reflect the
// passed list of `expectedURLs` and the canonical `url`.
//
// It returns the new canonical SoftwareURL and the new slice of aliases or an error if any.
func syncAliases( //nolint:cyclop // mostly error handling ifs
	gormdb *gorm.DB, software models.Software, canonicalURL string, expectedURLs []string,
) (*models.SoftwareURL, []models.SoftwareURL, error) {
	toRemove := []string{}          // Slice of SoftwareURL ids to remove from the database
	toAdd := []models.SoftwareURL{} // Slice of SoftwareURLs to add to the database

	// Map mirroring the state of SoftwareURLs for this software in the database,
	// keyed by url
	urlMap := map[string]models.SoftwareURL{}

	for _, alias := range software.Aliases {
		urlMap[alias.URL] = alias
	}

	for url, alias := range urlMap {
		if !slices.Contains(expectedURLs, url) {
			toRemove = append(toRemove, alias.ID)

			delete(urlMap, url)
		}
	}

	for _, url := range expectedURLs {
		_, exists := urlMap[url]
		if !exists {
			alias := models.SoftwareURL{ID: utils.UUIDv4(), URL: url, SoftwareID: software.ID}

			toAdd = append(toAdd, alias)
			urlMap[url] = alias
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

	updatedCanonicalURL := urlMap[canonicalURL]

	// Remove the canonical URL from the aliases, because it need to be its own
	// field. It was loaded previously together with the other aliases in Preload(),
	// because of limitation in gorm.
	delete(urlMap, canonicalURL)

	aliases := make([]models.SoftwareURL, 0, len(urlMap))
	for _, alias := range urlMap {
		aliases = append(aliases, alias)
	}

	return &updatedCanonicalURL, aliases, nil
}
