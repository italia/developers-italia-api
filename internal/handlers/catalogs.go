package handlers

import (
	"encoding/json"
	"errors"
	"net/url"
	"sort"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"github.com/italia/developers-italia-api/internal/handlers/general"
	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"
)

// rootCatalogID is the path parameter value that refers to the implicit root
// catalog (resources with catalog_id IS NULL).
const rootCatalogID = "∅"

type CatalogInterface interface { //nolint:interfacebloat
	GetCatalogs(ctx *fiber.Ctx) error
	GetCatalog(ctx *fiber.Ctx) error
	PostCatalog(ctx *fiber.Ctx) error
	PatchCatalog(ctx *fiber.Ctx) error
	DeleteCatalog(ctx *fiber.Ctx) error

	GetCatalogPublishers(ctx *fiber.Ctx) error
	PostCatalogPublisher(ctx *fiber.Ctx) error
	PatchCatalogPublisher(ctx *fiber.Ctx) error
	GetCatalogSoftware(ctx *fiber.Ctx) error
	PostCatalogSoftware(ctx *fiber.Ctx) error
	PatchCatalogSoftware(ctx *fiber.Ctx) error
}

type Catalog struct {
	db *gorm.DB
}

func NewCatalog(db *gorm.DB) *Catalog {
	return &Catalog{db: db}
}

// catalogScope returns a GORM scope that filters by catalog.
// nil catalog means root (catalog_id IS NULL).
func catalogScope(catalog *models.Catalog) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if catalog == nil {
			return db.Where("catalog_id IS NULL")
		}

		return db.Where("catalog_id = ?", catalog.ID)
	}
}

// GetCatalogs gets the list of all catalogs.
func (c *Catalog) GetCatalogs(ctx *fiber.Ctx) error {
	var catalogs []models.Catalog

	stmt, err := general.Clauses(ctx, c.db, "")
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Catalogs", err.Error())
	}

	if all := ctx.QueryBool("all", false); !all {
		stmt = stmt.Scopes(models.Active)
	}

	paginator := general.NewPaginator(ctx)

	result, cursor, err := paginator.Paginate(stmt, &catalogs)
	if err != nil {
		return common.Error(
			fiber.StatusUnprocessableEntity,
			"can't get Catalogs",
			"wrong cursor format in page[after] or page[before]",
		)
	}

	if result.Error != nil {
		return common.Error(
			fiber.StatusInternalServerError,
			"can't get Catalogs",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &catalogs, "links": general.PaginationLinks(cursor)})
}

// GetCatalog gets the catalog with the given id.
func (c *Catalog) GetCatalog(ctx *fiber.Ctx) error {
	id, _ := url.PathUnescape(ctx.Params("id"))

	catalog, err := c.resolveCatalog(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Catalog", "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Catalog", fiber.ErrInternalServerError.Message)
	}

	if catalog == nil {
		// Root catalog: return a synthetic representation.
		return common.Error(fiber.StatusNotFound, "can't get Catalog", "Catalog was not found")
	}

	return ctx.JSON(catalog)
}

// PostCatalog creates a new catalog.
func (c *Catalog) PostCatalog(ctx *fiber.Ctx) error {
	const errMsg = "can't create Catalog"

	request := new(common.CatalogPost)

	if err := common.ValidateRequestEntity(ctx, request, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	catalog := &models.Catalog{
		ID:            utils.UUIDv4(),
		Name:          request.Name,
		AlternativeID: request.AlternativeID,
		Active:        request.Active,
	}

	if err := c.db.Create(catalog).Error; err != nil {
		if field := common.DuplicateField(err); field != nil {
			detail := alreadyExists
			if *field != "" {
				detail = *field + " " + alreadyExists
			}

			return common.Error(fiber.StatusConflict, errMsg, detail)
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	return ctx.JSON(catalog)
}

// PatchCatalog updates the catalog with the given id.
func (c *Catalog) PatchCatalog(ctx *fiber.Ctx) error { //nolint:cyclop
	const errMsg = "can't update Catalog"

	catalogID, _ := url.PathUnescape(ctx.Params("id"))

	if catalogID == rootCatalogID {
		return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
	}

	catalog := models.Catalog{}

	if err := c.db.First(&catalog, "id = ? OR alternative_id = ?", catalogID, catalogID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	catalogJSON, err := json.Marshal(&catalog)
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

		updatedJSON, err = patch.Apply(catalogJSON)
		if err != nil {
			return common.Error(fiber.StatusUnprocessableEntity, errMsg, err.Error())
		}

	default:
		catalogReq := new(common.CatalogPatch)

		if err := common.ValidateRequestEntity(ctx, catalogReq, errMsg); err != nil {
			return err //nolint:wrapcheck
		}

		updatedJSON, err = jsonpatch.MergePatch(catalogJSON, ctx.Body())
		if err != nil {
			return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
		}
	}

	var updatedCatalog models.Catalog

	if err := json.Unmarshal(updatedJSON, &updatedCatalog); err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	updatedCatalog.ID = catalog.ID

	if err := c.db.Updates(&updatedCatalog).Error; err != nil {
		if field := common.DuplicateField(err); field != nil {
			detail := alreadyExists
			if *field != "" {
				detail = *field + " " + alreadyExists
			}

			return common.Error(fiber.StatusConflict, errMsg, detail)
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	return ctx.JSON(&updatedCatalog)
}

// DeleteCatalog deletes the catalog with the given id.
// Returns 409 if the catalog still has associated publishers or software.
func (c *Catalog) DeleteCatalog(ctx *fiber.Ctx) error {
	const errMsg = "can't delete Catalog"

	catalogID, _ := url.PathUnescape(ctx.Params("id"))

	if catalogID == rootCatalogID {
		return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
	}

	catalog := models.Catalog{}

	if err := c.db.First(&catalog, "id = ? OR alternative_id = ?", catalogID, catalogID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	var conflictErr error

	if err := c.db.Transaction(func(tran *gorm.DB) error {
		var publisherCount, softwareCount int64

		if err := tran.Model(&models.Publisher{}).
			Where("catalog_id = ?", catalog.ID).Count(&publisherCount).Error; err != nil {
			return err
		}

		if err := tran.Model(&models.Software{}).Where("catalog_id = ?", catalog.ID).Count(&softwareCount).Error; err != nil {
			return err
		}

		if publisherCount > 0 || softwareCount > 0 {
			conflictErr = common.Error(fiber.StatusConflict, errMsg, "Catalog still has associated publishers or software")

			return nil
		}

		return tran.Where("id = ?", catalog.ID).Delete(&models.Catalog{}).Error
	}); err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, "db error")
	}

	if conflictErr != nil {
		return conflictErr //nolint:wrapcheck
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

// GetCatalogPublishers lists publishers belonging to the given catalog.
func (c *Catalog) GetCatalogPublishers(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	catalog, err := c.resolveCatalog(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Publishers", "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Publishers", fiber.ErrInternalServerError.Message)
	}

	var publishers []models.Publisher

	stmt := c.db.Preload("CodeHosting").Scopes(catalogScope(catalog))

	stmt, err = general.Clauses(ctx, stmt, "")
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Publishers", err.Error())
	}

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
			"can't get Publishers",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &publishers, "links": general.PaginationLinks(cursor)})
}

// PostCatalogPublisher creates a publisher belonging to the given catalog.
// The catalog is resolved from the URL; any catalogId in the body is ignored.
func (c *Catalog) PostCatalogPublisher(ctx *fiber.Ctx) error { //nolint:cyclop
	const errMsg = "can't create Publisher"

	catalog, err := c.resolveCatalog(ctx.Params("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	request := new(common.PublisherPost)

	if err := common.ValidateRequestEntity(ctx, request, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	var catalogID *string
	if catalog != nil {
		catalogID = &catalog.ID
	}

	publisher := &models.Publisher{
		ID:            utils.UUIDv4(),
		CatalogID:     catalogID,
		Description:   request.Description,
		Email:         common.NormalizeEmail(request.Email),
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

	if err := c.db.Transaction(func(tran *gorm.DB) error {
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

	return ctx.JSON(publisher)
}

// PatchCatalogPublisher updates a publisher that belongs to the given catalog.
func (c *Catalog) PatchCatalogPublisher(ctx *fiber.Ctx) error { //nolint:cyclop,funlen,gocognit
	const errMsg = "can't update Publisher"

	catalog, err := c.resolveCatalog(ctx.Params("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	publisher := models.Publisher{}
	publisherID := ctx.Params("publisherId")

	if err := c.db.Preload("CodeHosting").
		First(&publisher, "id = ? or alternative_id = ?", publisherID, publisherID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Publisher was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	// Verify the publisher belongs to the resolved catalog.
	if catalog == nil {
		if publisher.CatalogID != nil {
			return common.Error(fiber.StatusNotFound, errMsg, "Publisher was not found")
		}
	} else if publisher.CatalogID == nil || *publisher.CatalogID != catalog.ID {
		return common.Error(fiber.StatusNotFound, errMsg, "Publisher was not found")
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

	// Prevent patches from changing the ID or catalog assignment.
	updatedPublisher.ID = publisher.ID
	updatedPublisher.CatalogID = publisher.CatalogID

	updatedPublisher.Email = common.NormalizeEmail(updatedPublisher.Email)

	expectedURLs := make([]string, 0, len(updatedPublisher.CodeHosting))
	for _, ch := range updatedPublisher.CodeHosting {
		expectedURLs = append(expectedURLs, common.NormalizeURL(ch.URL))
	}

	if err := c.db.Transaction(func(tran *gorm.DB) error { //nolint:dupl
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

	sort.Slice(publisher.CodeHosting, func(a int, b int) bool {
		return publisher.CodeHosting[a].URL < publisher.CodeHosting[b].URL
	})

	return ctx.JSON(&publisher)
}

// PostCatalogSoftware creates software belonging to the given catalog.
// The catalog is resolved from the URL; any catalogId in the body is ignored.
func (c *Catalog) PostCatalogSoftware(ctx *fiber.Ctx) error {
	const errMsg = "can't create Software"

	catalog, err := c.resolveCatalog(ctx.Params("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	softwareReq := new(common.SoftwarePost)

	if err := common.ValidateRequestEntity(ctx, softwareReq, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	var catalogID *string
	if catalog != nil {
		catalogID = &catalog.ID
	}

	aliases := []models.SoftwareURL{}
	for _, u := range softwareReq.Aliases {
		aliases = append(aliases, models.SoftwareURL{ID: utils.UUIDv4(), URL: common.NormalizeURL(u)})
	}

	url := models.SoftwareURL{ID: utils.UUIDv4(), URL: common.NormalizeURL(softwareReq.URL)}
	software := &models.Software{
		ID:            utils.UUIDv4(),
		URL:           url,
		SoftwareURLID: url.ID,
		CatalogID:     catalogID,
		Aliases:       aliases,
		PubliccodeYml: softwareReq.PubliccodeYml,
		Active:        softwareReq.Active,
		Vitality:      softwareReq.Vitality,
	}

	if err := c.db.Create(software).Error; err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	return ctx.JSON(software)
}

// PatchCatalogSoftware updates software that belongs to the given catalog.
func (c *Catalog) PatchCatalogSoftware(ctx *fiber.Ctx) error { //nolint:funlen,cyclop,gocognit
	const errMsg = "can't update Software"

	catalog, err := c.resolveCatalog(ctx.Params("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	software := models.Software{}

	if err := loadSoftware(c.db, &software, ctx.Params("softwareId")); err != nil {
		if errors.Is(err, errLoadNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	// Verify the software belongs to the resolved catalog.
	if catalog == nil {
		if software.CatalogID != nil {
			return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
		}
	} else if software.CatalogID == nil || *software.CatalogID != catalog.ID {
		return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
	}

	softwareJSON, err := json.Marshal(&software)
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

		updatedJSON, err = patch.Apply(softwareJSON)
		if err != nil {
			return common.Error(fiber.StatusUnprocessableEntity, errMsg, err.Error())
		}

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

	if err := json.Unmarshal(updatedJSON, &updatedSoftware); err != nil {
		return common.Error(fiber.StatusInternalServerError, errMsg, err.Error())
	}

	// Catalog assignment is immutable via patch; keep the value from the URL.
	updatedSoftware.CatalogID = software.CatalogID

	updatedSoftware.URL.URL = common.NormalizeURL(updatedSoftware.URL.URL)

	expectedAliases := make([]string, 0, len(updatedSoftware.Aliases))
	for _, alias := range updatedSoftware.Aliases {
		expectedAliases = append(expectedAliases, common.NormalizeURL(alias.URL))
	}

	if err := c.db.Transaction(func(tran *gorm.DB) error {
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

		updatedSoftware.SoftwareURLID = updatedURL.ID
		updatedSoftware.URL = *updatedURL

		// Set Aliases to a zero value, so it's not touched by gorm's Update(),
		// because we handle the alias manually.
		updatedSoftware.Aliases = []models.SoftwareURL{}

		if err := tran.Updates(&updatedSoftware).Error; err != nil {
			return err
		}

		updatedSoftware.Aliases = aliases

		return nil
	}); err != nil {
		if field := common.DuplicateField(err); field != nil {
			detail := alreadyExists
			if *field != "" {
				detail = *field + " " + alreadyExists
			}

			return common.Error(fiber.StatusConflict, errMsg, detail)
		}

		//nolint:wrapcheck
		return err
	}

	sort.Slice(updatedSoftware.Aliases, func(a int, b int) bool {
		return updatedSoftware.Aliases[a].URL < updatedSoftware.Aliases[b].URL
	})

	return ctx.JSON(&updatedSoftware)
}

// GetCatalogSoftware lists software belonging to the given catalog.
func (c *Catalog) GetCatalogSoftware(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	catalog, err := c.resolveCatalog(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, "can't get Software", "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, "can't get Software", fiber.ErrInternalServerError.Message)
	}

	var software []models.Software

	stmt := c.db.Preload("URL").Preload("Aliases").Scopes(catalogScope(catalog))

	stmt, err = general.Clauses(ctx, stmt, "")
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Software", err.Error())
	}

	if all := ctx.QueryBool("all", false); !all {
		stmt = stmt.Scopes(models.Active)
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

	return ctx.JSON(fiber.Map{"data": &software, "links": general.PaginationLinks(cursor)})
}

// resolveCatalog looks up a catalog by id or alternativeId.
// If id is rootCatalogID and no catalog with that id exists, it returns nil
// (meaning: filter by catalog_id IS NULL).
func (c *Catalog) resolveCatalog(rawID string) (*models.Catalog, error) {
	catalogID, err := url.PathUnescape(rawID)
	if err != nil {
		catalogID = rawID
	}

	var catalog models.Catalog

	dbErr := c.db.First(&catalog, "id = ? OR alternative_id = ?", catalogID, catalogID).Error
	if dbErr == nil {
		return &catalog, nil
	}

	if errors.Is(dbErr, gorm.ErrRecordNotFound) && catalogID == rootCatalogID {
		return nil, nil //nolint:nilnil
	}

	return nil, dbErr
}
