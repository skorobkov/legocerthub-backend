package sqlite

import (
	"context"
	"errors"
	"legocerthub-backend/pkg/domain/acme_accounts"
	"legocerthub-backend/pkg/utils/acme_utils"
)

// accountPayloadToDb turns the client payload into a db object
func nameDescAccountPayloadToDb(payload acme_accounts.NameDescPayload) (accountDb, error) {
	var dbObj accountDb
	var err error

	// payload ID should never be missing at this point, regardless error if it somehow
	//  is to avoid nil pointer dereference
	if payload.ID == nil {
		err = errors.New("id missing in payload")
		return accountDb{}, err
	}
	dbObj.id = *payload.ID

	dbObj.name = *payload.Name

	dbObj.description = stringToNullString(payload.Description)

	return dbObj, nil
}

// putExistingAccountNameDesc only updates the name and desc in the database
// refactor to more generic for anything that can be updated??
func (storage *Storage) PutNameDescAccount(payload acme_accounts.NameDescPayload) error {
	// Load payload into db obj
	accountDb, err := nameDescAccountPayloadToDb(payload)
	if err != nil {
		return err
	}

	// database update

	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `
	UPDATE
		acme_accounts
	SET
		name = $1,
		description = $2
	WHERE
		id = $3`

	_, err = storage.Db.ExecContext(ctx, query,
		accountDb.name,
		accountDb.description,
		accountDb.id)
	if err != nil {
		return err
	}

	// TODO: Handle 0 rows updated.
	return nil
}

// dbPutExistingAccount overwrites an existing acme account using specified data
// certain fields cannot be updated, per rfc8555
func (storage *Storage) PutExistingAccount(payload acme_accounts.AccountPayload) error {
	// Load payload into db obj
	accountDb, err := accountPayloadToDb(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `
	UPDATE
		acme_accounts
	SET
		name = $1,
		description = $2,
		email = $3,
		accepted_tos = case when $4 is null then accepted_tos else $4 end,
		updated_at = $5
	WHERE
		id = $6`

	_, err = storage.Db.ExecContext(ctx, query,
		accountDb.name,
		accountDb.description,
		accountDb.email,
		accountDb.acceptedTos,
		accountDb.updatedAt,
		accountDb.id)
	if err != nil {
		return err
	}

	// TODO: Handle 0 rows updated.
	return nil
}

// putLEAccountInfo populates an account with data that is returned by LE when
//  an account is POSTed to
func (storage *Storage) PutLEAccountInfo(id int, response acme_utils.AcmeAccountResponse) error {
	// Load id and response into db obj
	accountDb := acmeAccountToDb(id, response)

	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `
	UPDATE
		acme_accounts
	SET
		status = $1,
		email = $2,
		created_at = $3,
		updated_at = $4,
		kid = $5
	WHERE
		id = $6`

	_, err := storage.Db.ExecContext(ctx, query,
		accountDb.status,
		accountDb.email,
		accountDb.createdAt,
		accountDb.updatedAt,
		accountDb.kid,
		accountDb.id)
	if err != nil {
		return err
	}

	// TODO: Handle 0 rows updated.
	return nil
}