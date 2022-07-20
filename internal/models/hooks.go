package models

import (
	"log"

	"github.com/gofiber/fiber/v2/utils"
	"github.com/italia/developers-italia-api/internal/common"
	"gorm.io/gorm"
)

var EventChan = make(chan Event) //nolint:gochecknoglobals

func (p Publisher) AfterCreate(trx *gorm.DB) error {
	event := Event{
		ID:         utils.UUIDv4(),
		Type:       common.EventTypeCreate,
		EntityType: p.TableName(),
		EntityID:   p.UUID(),
	}

	if err := trx.Create(&event).Error; err != nil {
		return err
	}

	sendNonBlock(event)

	return nil
}

func (s Software) AfterCreate(trx *gorm.DB) error {
	event := Event{
		ID:         utils.UUIDv4(),
		Type:       common.EventTypeCreate,
		EntityType: s.TableName(),
		EntityID:   s.UUID(),
	}

	if err := trx.Create(&event).Error; err != nil {
		return err
	}

	sendNonBlock(event)

	return nil
}

func (p Publisher) AfterUpdate(trx *gorm.DB) error {
	event := Event{
		ID:         utils.UUIDv4(),
		Type:       common.EventTypeUpdate,
		EntityType: p.TableName(),
		EntityID:   p.UUID(),
	}

	if err := trx.Create(&event).Error; err != nil {
		return err
	}

	sendNonBlock(event)

	return nil
}

func (s Software) AfterUpdate(trx *gorm.DB) error {
	event := Event{
		ID:         utils.UUIDv4(),
		Type:       common.EventTypeUpdate,
		EntityType: s.TableName(),
		EntityID:   s.UUID(),
	}

	if err := trx.Create(&event).Error; err != nil {
		return err
	}

	sendNonBlock(event)

	return nil
}

func (p Publisher) AfterDelete(trx *gorm.DB) error {
	event := Event{
		ID:         utils.UUIDv4(),
		Type:       common.EventTypeDelete,
		EntityType: p.TableName(),
		EntityID:   p.UUID(),
	}

	if err := trx.Create(&event).Error; err != nil {
		return err
	}

	sendNonBlock(event)

	return nil
}

func (s Software) AfterDelete(trx *gorm.DB) error {
	event := Event{
		ID:         utils.UUIDv4(),
		Type:       common.EventTypeDelete,
		EntityType: s.TableName(),
		EntityID:   s.UUID(),
	}

	if err := trx.Create(&event).Error; err != nil {
		return err
	}

	sendNonBlock(event)

	return nil
}

func sendNonBlock(event Event) {
	select {
	case EventChan <- event:
	default:
		log.Printf("can't send event %v to channel\n", event)
	}
}
