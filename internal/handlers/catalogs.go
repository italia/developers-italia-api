package handlers

import (
	"errors"
	"net/url"
	"sort"

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

// isRoot reports whether the given catalog represents the implicit root.
// A nil catalog (no row found at the ∅ sentinel) and a row whose
// alternativeId is ∅ are both treated as root: their resources have
// catalog_id IS NULL.
func isRoot(catalog *models.Catalog) bool {
	if catalog == nil {
		return true
	}

	return catalog.AlternativeID != nil && *catalog.AlternativeID == rootCatalogID
}

// catalogScope returns a GORM scope that filters by catalog.
// Root catalog (implicit or materialized as the ∅ alias) means catalog_id IS NULL.
func catalogScope(catalog *models.Catalog) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if isRoot(catalog) {
			return db.Where("catalog_id IS NULL")
		}

		return db.Where("catalog_id = ?", catalog.ID)
	}
}

// GetCatalogs gets the list of all catalogs.
func (c *Catalog) GetCatalogs(ctx *fiber.Ctx) error {
	var catalogs []models.Catalog

	stmt, err := general.Clauses(ctx, c.db.Preload("Sources"), "")
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Catalogs", err.Error())
	}

	if all := ctx.QueryBool("all", false); !all {
		stmt = stmt.Scopes(models.Active)
	}

	paginator, err := general.NewPaginator(ctx)
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Catalogs", err.Error())
	}

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

	return ctx.JSON(fiber.Map{"data": &catalogs, "links": general.NewPaginationLinks(ctx.Queries(), cursor)})
}

// GetCatalog gets the catalog with the given id.
func (c *Catalog) GetCatalog(ctx *fiber.Ctx) error {
	id, _ := url.PathUnescape(ctx.Params("id"))

	catalog, err := resolveCatalog(c.db, id, "Sources")
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
// When alternativeId is "∅" the new row materializes the root catalog: it
// holds the configuration (sources are not allowed; resources are still
// addressed via catalog_id IS NULL). For any other catalog at least one
// source is required.
func (c *Catalog) PostCatalog(ctx *fiber.Ctx) error {
	const errMsg = "can't create Catalog"

	request := new(common.CatalogPost)

	if err := common.ValidateRequestEntity(ctx, request, errMsg); err != nil {
		return err //nolint:wrapcheck
	}

	asRoot := request.AlternativeID != nil && *request.AlternativeID == rootCatalogID

	if asRoot && len(request.Sources) > 0 {
		return common.Error(fiber.StatusUnprocessableEntity, errMsg,
			"sources are not allowed on the root catalog")
	}

	if !asRoot && len(request.Sources) == 0 {
		return common.Error(fiber.StatusUnprocessableEntity, errMsg, "sources is required")
	}

	sources := buildSources(request.Sources)

	catalog := &models.Catalog{
		ID:                  utils.UUIDv4(),
		Name:                request.Name,
		AlternativeID:       request.AlternativeID,
		Active:              request.Active,
		Scopes:              request.Scopes,
		PublishersNamespace: request.PublishersNamespace,
		Sources:             sources,
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
func (c *Catalog) PatchCatalog(ctx *fiber.Ctx) error { //nolint:cyclop,funlen
	const errMsg = "can't update Catalog"

	catalogID, _ := url.PathUnescape(ctx.Params("id"))

	resolved, err := resolveCatalog(c.db, catalogID, "Sources")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	if resolved == nil {
		return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
	}

	catalog := *resolved

	contentType := ctx.Get(fiber.HeaderContentType)
	if contentType != common.ContentTypeJSONPatch {
		if err := common.ValidateRequestEntity(ctx, new(common.CatalogPatch), errMsg); err != nil {
			return err //nolint:wrapcheck
		}
	}

	updatedCatalog, patchErr := common.ApplyPatch(&catalog, contentType, ctx.Body())
	if patchErr != nil {
		return common.Error(patchErr.Code, errMsg, patchErr.Error())
	}

	updatedCatalog.ID = catalog.ID

	if isRoot(&catalog) {
		if updatedCatalog.AlternativeID == nil || *updatedCatalog.AlternativeID != rootCatalogID {
			return common.Error(fiber.StatusUnprocessableEntity, errMsg,
				"alternativeId on the root catalog cannot be changed")
		}
	}

	sourcesInput := make([]common.SourceInput, 0, len(updatedCatalog.Sources))
	for _, src := range updatedCatalog.Sources {
		sourcesInput = append(sourcesInput, common.SourceInput{
			URL:    src.URL,
			Driver: src.Driver,
			Args:   src.Args,
		})
	}

	if isRoot(&catalog) {
		if len(sourcesInput) > 0 {
			return common.Error(fiber.StatusUnprocessableEntity, errMsg,
				"sources are not allowed on the root catalog")
		}
	} else if len(sourcesInput) == 0 {
		return common.Error(fiber.StatusUnprocessableEntity, errMsg, "sources must not be empty")
	}

	if err := c.db.Transaction(func(tran *gorm.DB) error {
		sources, err := syncSources(tran, catalog, sourcesInput)
		if err != nil {
			return err
		}

		updatedCatalog.Sources = nil

		if err := tran.Updates(&updatedCatalog).Error; err != nil {
			return err
		}

		updatedCatalog.Sources = sources

		return nil
	}); err != nil {
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
// On the root (∅) the count of attached resources is taken from rows with
// catalog_id IS NULL, since root resources are never tied to the row's UUID.
func (c *Catalog) DeleteCatalog(ctx *fiber.Ctx) error { //nolint:cyclop
	const errMsg = "can't delete Catalog"

	catalogID, _ := url.PathUnescape(ctx.Params("id"))

	resolved, err := resolveCatalog(c.db, catalogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
		}

		return common.Error(fiber.StatusInternalServerError, errMsg, fiber.ErrInternalServerError.Message)
	}

	if resolved == nil {
		return common.Error(fiber.StatusNotFound, errMsg, "Catalog was not found")
	}

	catalog := *resolved

	var conflictErr error

	if err := c.db.Transaction(func(tran *gorm.DB) error {
		var publisherCount, softwareCount int64

		pubScope := tran.Model(&models.Publisher{}).Scopes(catalogScope(&catalog))
		if err := pubScope.Count(&publisherCount).Error; err != nil {
			return err
		}

		swScope := tran.Model(&models.Software{}).Scopes(catalogScope(&catalog))
		if err := swScope.Count(&softwareCount).Error; err != nil {
			return err
		}

		if publisherCount > 0 || softwareCount > 0 {
			conflictErr = common.Error(fiber.StatusConflict, errMsg, "Catalog still has associated publishers or software")

			return nil
		}

		if err := tran.Where("catalog_id = ?", catalog.ID).Delete(&models.CatalogSource{}).Error; err != nil {
			return err
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

	catalog, err := resolveCatalog(c.db, id)
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
			"can't get Publishers",
			fiber.ErrInternalServerError.Message,
		)
	}

	return ctx.JSON(fiber.Map{"data": &publishers, "links": general.NewPaginationLinks(ctx.Queries(), cursor)})
}

// PostCatalogPublisher creates a publisher belonging to the given catalog.
// The catalog is resolved from the URL; any catalogId in the body is ignored.
func (c *Catalog) PostCatalogPublisher(ctx *fiber.Ctx) error { //nolint:cyclop
	const errMsg = "can't create Publisher"

	catalog, err := resolveCatalog(c.db, ctx.Params("id"))
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
	if !isRoot(catalog) {
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

	catalog, err := resolveCatalog(c.db, ctx.Params("id"))
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
	if isRoot(catalog) {
		if publisher.CatalogID != nil {
			return common.Error(fiber.StatusNotFound, errMsg, "Publisher was not found")
		}
	} else if publisher.CatalogID == nil || *publisher.CatalogID != catalog.ID {
		return common.Error(fiber.StatusNotFound, errMsg, "Publisher was not found")
	}

	contentType := ctx.Get(fiber.HeaderContentType)
	if contentType != common.ContentTypeJSONPatch {
		if err := common.ValidateRequestEntity(ctx, new(common.PublisherPatch), errMsg); err != nil {
			return err //nolint:wrapcheck
		}
	}

	updatedPublisher, patchErr := common.ApplyPatch(&publisher, contentType, ctx.Body())
	if patchErr != nil {
		return common.Error(patchErr.Code, errMsg, patchErr.Error())
	}

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

	catalog, err := resolveCatalog(c.db, ctx.Params("id"))
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
	if !isRoot(catalog) {
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
func (c *Catalog) PatchCatalogSoftware(ctx *fiber.Ctx) error { //nolint:funlen,cyclop
	const errMsg = "can't update Software"

	catalog, err := resolveCatalog(c.db, ctx.Params("id"))
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
	if isRoot(catalog) {
		if software.CatalogID != nil {
			return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
		}
	} else if software.CatalogID == nil || *software.CatalogID != catalog.ID {
		return common.Error(fiber.StatusNotFound, errMsg, "Software was not found")
	}

	contentType := ctx.Get(fiber.HeaderContentType)
	if contentType != common.ContentTypeJSONPatch {
		if err := common.ValidateRequestEntity(ctx, &common.SoftwarePatch{}, errMsg); err != nil {
			return err //nolint:wrapcheck
		}
	}

	updatedSoftware, patchErr := common.ApplyPatch(&software, contentType, ctx.Body())
	if patchErr != nil {
		return common.Error(patchErr.Code, errMsg, patchErr.Error())
	}

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
//
//nolint:cyclop // keeping request handling inline is clearer here
func (c *Catalog) GetCatalogSoftware(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	catalog, err := resolveCatalog(c.db, id)
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

	if urlFilter := common.NormalizeURL(ctx.Query("url", "")); urlFilter != "" {
		var softwareURL models.SoftwareURL

		if e := c.db.First(&softwareURL, "url = ?", urlFilter).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return ctx.JSON(fiber.Map{"data": []any{}, "links": general.PaginationLinks{}})
			}

			return common.Error(
				fiber.StatusInternalServerError, "can't get Software", fiber.ErrInternalServerError.Message,
			)
		}

		stmt = stmt.Where("id = ?", softwareURL.SoftwareID)
	}

	if all := ctx.QueryBool("all", false); !all {
		stmt = stmt.Scopes(models.Active)
	}

	paginator, err := general.NewPaginator(ctx)
	if err != nil {
		return common.Error(fiber.StatusUnprocessableEntity, "can't get Software", err.Error())
	}

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

	return ctx.JSON(fiber.Map{"data": &software, "links": general.NewPaginationLinks(ctx.Queries(), cursor)})
}

// buildSources converts SourceInput slice to CatalogSource models.
func buildSources(inputs []common.SourceInput) []models.CatalogSource {
	sources := make([]models.CatalogSource, 0, len(inputs))

	for _, inp := range inputs {
		sources = append(sources, models.CatalogSource{
			ID:     utils.UUIDv4(),
			Driver: inp.Driver,
			URL:    common.NormalizeURL(inp.URL),
			Args:   inp.Args,
		})
	}

	return sources
}

// syncSources brings the catalog_sources table in line with the desired state.
// Sources are matched by URL; removed if absent, added if new.
func syncSources( //nolint:cyclop,funlen
	gormdb *gorm.DB,
	catalog models.Catalog,
	desired []common.SourceInput,
) ([]models.CatalogSource, error) {
	toRemove := []string{}
	toAdd := []models.CatalogSource{}
	toUpdate := []models.CatalogSource{}

	urlMap := map[string]models.CatalogSource{}
	for _, src := range catalog.Sources {
		urlMap[src.URL] = src
	}

	desiredSet := map[string]common.SourceInput{}
	for _, inp := range desired {
		desiredSet[common.NormalizeURL(inp.URL)] = inp
	}

	for srcURL, src := range urlMap {
		if _, ok := desiredSet[srcURL]; !ok {
			toRemove = append(toRemove, src.ID)

			delete(urlMap, srcURL)
		}
	}

	for srcURL, inp := range desiredSet {
		if existing, ok := urlMap[srcURL]; ok {
			changed := false

			if inp.Driver != nil && (existing.Driver == nil || *existing.Driver != *inp.Driver) {
				existing.Driver = inp.Driver
				changed = true
			}

			if inp.Args != nil {
				existing.Args = inp.Args
				changed = true
			}

			if changed {
				toUpdate = append(toUpdate, existing)
				urlMap[srcURL] = existing
			}
		} else {
			src := models.CatalogSource{
				ID:        utils.UUIDv4(),
				Driver:    inp.Driver,
				URL:       common.NormalizeURL(srcURL),
				Args:      inp.Args,
				CatalogID: catalog.ID,
			}
			toAdd = append(toAdd, src)
			urlMap[srcURL] = src
		}
	}

	if len(toRemove) > 0 {
		if err := gormdb.Delete(&models.CatalogSource{}, toRemove).Error; err != nil {
			return nil, err
		}
	}

	if len(toAdd) > 0 {
		if err := gormdb.Create(toAdd).Error; err != nil {
			return nil, err
		}
	}

	for _, src := range toUpdate {
		if err := gormdb.Save(&src).Error; err != nil {
			return nil, err
		}
	}

	ret := make([]models.CatalogSource, 0, len(urlMap))
	for _, src := range urlMap {
		ret = append(ret, src)
	}

	return ret, nil
}

// resolveCatalog looks up a catalog by id or alternativeId.
// If id is rootCatalogID and no catalog with that id exists, it returns nil
// (meaning: filter by catalog_id IS NULL).
// Optional preloads (e.g. "Sources") are applied to the query.
func resolveCatalog(gormdb *gorm.DB, rawID string, preloads ...string) (*models.Catalog, error) {
	catalogID, err := url.PathUnescape(rawID)
	if err != nil {
		catalogID = rawID
	}

	var catalog models.Catalog

	stmt := gormdb
	for _, p := range preloads {
		stmt = stmt.Preload(p)
	}

	dbErr := stmt.First(&catalog, "id = ? OR alternative_id = ?", catalogID, catalogID).Error
	if dbErr == nil {
		return &catalog, nil
	}

	if errors.Is(dbErr, gorm.ErrRecordNotFound) && catalogID == rootCatalogID {
		return nil, nil //nolint:nilnil
	}

	return nil, dbErr
}
