package sqlite

import (
	"context"
	"legocerthub-backend/pkg/domain/certificates"
	"legocerthub-backend/pkg/domain/orders"
	"legocerthub-backend/pkg/domain/private_keys"
)

// accountDbToAcc turns the database representation of a certificate into a Certificate
func (orderDb *orderDb) orderDbToOrder() (order orders.Order, err error) {
	// convert embedded private key db
	var finalKey = new(private_keys.Key)
	if orderDb.finalizedKey != nil && orderDb.finalizedKey.id.Valid {
		*finalKey, err = orderDb.finalizedKey.keyDbToKey()
		if err != nil {
			return orders.Order{}, err
		}
	} else {
		finalKey = nil
	}

	// convert embedded cert db
	var cert = new(certificates.Certificate)
	if orderDb.certificate != nil && orderDb.certificate.id.Valid {
		*cert, err = orderDb.certificate.certDbToCert()
		if err != nil {
			return orders.Order{}, err
		}
	} else {
		cert = nil
	}

	return orders.Order{
		// omit account, not needed
		ID:             nullInt32ToInt(orderDb.id),
		Certificate:    cert,
		Location:       nullStringToString(orderDb.location),
		Status:         nullStringToString(orderDb.status),
		Error:          nullStringToAcmeError(orderDb.err),
		Expires:        nullInt32ToInt(orderDb.expires),
		DnsIdentifiers: commaNullStringToSlice(orderDb.dnsIdentifiers),
		Authorizations: commaNullStringToSlice(orderDb.authorizations),
		Finalize:       nullStringToString(orderDb.finalize),
		FinalizedKey:   finalKey,
		CertificateUrl: nullStringToString(orderDb.certificateUrl),
		CreatedAt:      nullInt32ToInt(orderDb.createdAt),
		UpdatedAt:      nullInt32ToInt(orderDb.updatedAt),
	}, nil
}

// GetOneOrder fetches a specific Order by ID
func (store *Storage) GetOneOrder(orderId int) (order orders.Order, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), store.Timeout)
	defer cancel()

	query := `
	SELECT
		ao.id, ao.acme_location, ao.status, ao.error, ao.expires, ao.dns_identifiers, ao.authorizations, ao.finalize, 
		ao.certificate_url, ao.created_at, ao.updated_at,
		pk.id, pk.name,
		c.id, c.name
	FROM
		acme_orders ao
		LEFT JOIN private_keys pk on (ao.finalized_key_id = pk.id)
		LEFT JOIN certificates c on (ao.certificate_id = c.id)
	WHERE
		ao.id = $1
	ORDER BY
		expires DESC
	`

	row := store.Db.QueryRowContext(ctx, query, orderId)

	var orderDb orderDb
	// initialize keyDb and certDb pointer (or nil deref)
	orderDb.finalizedKey = new(keyDb)
	orderDb.certificate = new(certificateDb)

	err = row.Scan(
		&orderDb.id,
		&orderDb.location,
		&orderDb.status,
		&orderDb.err,
		&orderDb.expires,
		&orderDb.dnsIdentifiers,
		&orderDb.authorizations,
		&orderDb.finalize,
		&orderDb.certificateUrl,
		&orderDb.createdAt,
		&orderDb.updatedAt,
		&orderDb.finalizedKey.id,
		&orderDb.finalizedKey.name,
		&orderDb.certificate.id,
		&orderDb.certificate.name,
	)
	if err != nil {
		return orders.Order{}, err
	}

	order, err = orderDb.orderDbToOrder()
	if err != nil {
		return orders.Order{}, err
	}

	return order, nil
}

// GetCertOrders fetches all of the orders for a specified certificate ID
func (store *Storage) GetCertOrders(certId int) (orders []orders.Order, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), store.Timeout)
	defer cancel()

	query := `
	SELECT
		ao.id, ao.acme_location, ao.status, ao.error, ao.expires, ao.dns_identifiers, ao.authorizations, ao.finalize, 
		ao.certificate_url, ao.created_at, ao.updated_at,
		pk.id, pk.name
	FROM
		acme_orders ao
		LEFT JOIN private_keys pk on (ao.finalized_key_id = pk.id)
	WHERE
		ao.certificate_id = $1
	ORDER BY
		expires DESC
	`

	rows, err := store.Db.QueryContext(ctx, query, certId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var oneOrder orderDb
		// initialize keyDb pointer (or nil deref)
		oneOrder.finalizedKey = new(keyDb)
		err = rows.Scan(
			&oneOrder.id,
			&oneOrder.location,
			&oneOrder.status,
			&oneOrder.err,
			&oneOrder.expires,
			&oneOrder.dnsIdentifiers,
			&oneOrder.authorizations,
			&oneOrder.finalize,
			&oneOrder.certificateUrl,
			&oneOrder.createdAt,
			&oneOrder.updatedAt,
			&oneOrder.finalizedKey.id,
			&oneOrder.finalizedKey.name,
		)
		if err != nil {
			return nil, err
		}

		convertedOrder, err := oneOrder.orderDbToOrder()
		if err != nil {
			return nil, err
		}

		orders = append(orders, convertedOrder)
	}

	return orders, nil
}