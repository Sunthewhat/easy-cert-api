package participantmodel

import (
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/shared/model"
)

func Revoke(id string) (*model.Participant, error) {
	// Get the participant by ID
	participant, err := common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	// Update the isrevoke field to true
	_, err = common.Gorm.Participant.Where(common.Gorm.Participant.ID.Eq(id)).Update(common.Gorm.Participant.Isrevoke, true)
	if err != nil {
		return nil, err
	}

	// Return the updated participant
	participant.Isrevoke = true
	return participant, nil
}
