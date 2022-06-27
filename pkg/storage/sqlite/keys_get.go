package sqlite

import (
	"context"
	"legocerthub-backend/pkg/domain/private_keys"
	"legocerthub-backend/pkg/domain/private_keys/key_crypto"
)

// KeyDbToKey translates the db object into the object the key service expects
func (keyDb *keyDb) keyDbToKey() (private_keys.Key, error) {
	var algorithm = new(key_crypto.Algorithm)
	var err error

	// if there is an algorithm value, specify the algorithm
	if keyDb.algorithmValue.Valid {
		*algorithm, err = key_crypto.AlgorithmByValue(keyDb.algorithmValue.String)
		if err != nil {
			return private_keys.Key{}, err
		}
	} else {
		algorithm = nil
	}

	return private_keys.Key{
		ID:          keyDb.id,
		Name:        keyDb.name.String,
		Description: keyDb.description.String,
		Algorithm:   algorithm,
		Pem:         keyDb.pem.String,
		ApiKey:      keyDb.apiKey,
		CreatedAt:   keyDb.createdAt,
		UpdatedAt:   keyDb.updatedAt,
	}, nil
}

// dbGetAllPrivateKeys writes information about all private keys to json
func (storage Storage) GetAllKeys() ([]private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `SELECT id, name, description, algorithm
	FROM private_keys ORDER BY id`

	rows, err := storage.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allKeys []private_keys.Key
	for rows.Next() {
		var oneKey keyDb
		err = rows.Scan(
			&oneKey.id,
			&oneKey.name,
			&oneKey.description,
			&oneKey.algorithmValue,
		)
		if err != nil {
			return nil, err
		}

		convertedKey, err := oneKey.keyDbToKey()
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, convertedKey)
	}

	return allKeys, nil
}

// GetOneKeyById returns a key based on its unique id
func (storage *Storage) GetOneKeyById(id int) (private_keys.Key, error) {
	return storage.getOneKey(id, "")
}

// GetOneKeyByName returns a key based on its unique name
func (storage *Storage) GetOneKeyByName(name string) (private_keys.Key, error) {
	return storage.getOneKey(-1, name)
}

// dbGetOneKey returns a key from the db based on unique id or unique name
func (storage Storage) getOneKey(id int, name string) (private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	query := `SELECT id, name, description, algorithm, pem, api_key, created_at, updated_at
	FROM private_keys
	WHERE id = $1 OR name = $2
	ORDER BY id`

	row := storage.Db.QueryRowContext(ctx, query, id, name)

	var oneKeyDb keyDb
	err := row.Scan(
		&oneKeyDb.id,
		&oneKeyDb.name,
		&oneKeyDb.description,
		&oneKeyDb.algorithmValue,
		&oneKeyDb.pem,
		&oneKeyDb.apiKey,
		&oneKeyDb.createdAt,
		&oneKeyDb.updatedAt,
	)

	if err != nil {
		return private_keys.Key{}, err
	}

	convertedKey, err := oneKeyDb.keyDbToKey()
	if err != nil {
		return private_keys.Key{}, err
	}

	return convertedKey, nil
}

// GetAvailableKeys returns a slice of private keys that exist but are not already associated
// with a known ACME account or certificate
func (storage *Storage) GetAvailableKeys() ([]private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storage.Timeout)
	defer cancel()

	// TODO - Once certs are added, need to check that table as well for keys in use
	query := `
		SELECT pk.id, pk.name, pk.description, pk.algorithm
		FROM
		  private_keys pk
		WHERE
			NOT EXISTS(
				SELECT
					aa.private_key_id
				FROM
					acme_accounts aa
				WHERE
					pk.id = aa.private_key_id
			)
	`

	rows, err := storage.Db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var availableKeys []private_keys.Key
	for rows.Next() {
		var oneKey keyDb

		err = rows.Scan(
			&oneKey.id,
			&oneKey.name,
			&oneKey.description,
			&oneKey.algorithmValue,
		)
		if err != nil {
			return nil, err
		}

		convertedKey, err := oneKey.keyDbToKey()
		if err != nil {
			return nil, err
		}

		availableKeys = append(availableKeys, convertedKey)
	}

	return availableKeys, nil
}
