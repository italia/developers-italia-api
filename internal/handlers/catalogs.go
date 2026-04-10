package handlers

import (
	"encoding/json"
	"errors"
	"net/url"

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

type CatalogInterface interface {
	GetCatalogs(ctx *fiber.Ctx) error
	GetCatalog(ctx *fiber.Ctx) error
	PostCatalog(ctx *fiber.Ctx) error
	PatchCatalog(ctx *fiber.Ctx) error
	DeleteCatalog(ctx *fiber.Ctx) error

	GetCatalogPublishers(ctx *fiber.Ctx) error
	GetCatalogSoftware(ctx *fiber.Ctx) error
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
