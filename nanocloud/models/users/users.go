/*
 * Nanocloud Community, a comprehensive platform to turn any application
 * into a cloud solution.
 *
 * Copyright (C) 2015 Nanocloud Software
 *
 * This file is part of Nanocloud community.
 *
 * Nanocloud community is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * Nanocloud community is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package users

import (
	errors "errors"
	"database/sql"

	"github.com/Nanocloud/community/nanocloud/connectors/db"
	log "github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	UserNotFound       = errors.New("user not found")
	InvalidCredentials = errors.New("invalid credentials")
	UserDisabled       = errors.New("user disabled")
	UserDuplicated     = errors.New("user duplicated")
	UserNotCreated     = errors.New("user not created")
)

func GetUserFromEmailPassword(email, password string) (*User, error) {
	if len(email) < 1 || len(password) < 1 {
		return nil, UserNotFound
	}

	rows, err := db.Query(
		`SELECT id, activated,
		email, password,
		first_name, last_name,
		is_admin, expiration_date
		FROM users
		WHERE email = $1::varchar AND (expiration_date IS NULL OR expiration_date > current_timestamp)`,
		email,
	)
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, UserNotFound
	}

	var user User
	var passwordHash string
	rows.Scan(
		&user.Id, &user.Activated,
		&user.Email, &passwordHash,
		&user.FirstName, &user.LastName,
		&user.IsAdmin, &user.ExpDate,
	)
	rows.Close()

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

	if err != nil {
		return nil, InvalidCredentials
	}

	if !user.Activated {
		return nil, UserDisabled
	}

	return &user, nil
}

func FindUsers() ([]*User, error) {
	rows, err := db.Query(
		`SELECT id, first_name, last_name, email,
			is_admin, activated, extract(epoch from signup_date), expiration_date
		FROM users`,
	)
	if err != nil {
		return nil, err
	}

	var users []*User
	var timestamp float64

	defer rows.Close()
	for rows.Next() {
		user := User{}

		rows.Scan(
			&user.Id,
			&user.FirstName, &user.LastName,
			&user.Email,
			&user.IsAdmin,
			&user.Activated,
			&timestamp,
			&user.ExpDate,
		)
		// javascript time is in millisecond not in second
		user.SignupDate = int(1000 * timestamp)
		users = append(users, &user)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return users, nil
}

func UserExists(id string) (bool, error) {
	rows, err := db.Query(
		`SELECT id
		FROM users
		WHERE id = $1::varchar`,
		id)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		return true, nil
	}
	return false, nil
}

func DisableUser(id string) error {
	rows, err := db.Query(
		`UPDATE users
		SET activated = false
		WHERE id = $1::varchar`,
		id)
	if err != nil {
		rows.Close()
	}
	return err
}

func CreateUser(
	activated bool,
	email string,
	firstName string,
	lastName string,
	password string,
	isAdmin bool,
	expirationDate string,
) (*User, error) {
	id := uuid.NewV4().String()

	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`INSERT INTO users
	(id, email, activated,
	first_name, last_name,
	password, is_admin)
	VALUES(
	  $1::varchar, $2::varchar, $3::bool,
	  $4::varchar, $5::varchar,
	  $6::varchar, $7::bool
	  )
	RETURNING id, email, activated,
	first_name, last_name,
	is_admin`,
		id, email, activated,
		firstName, lastName,
		pass, isAdmin)

	if expirationDate != "" {
			expirationDate = expirationDate + "days"
			expirationDate, err := db.Query(
					`SELECT current_timestamp + interval $1`,
			expirationDate)
			if err != nil {
					defer expirationDate.Close()
			}
			if !expirationDate.Next() {
					return nil, UserNotCreated
			}
			expirationDate.Scan(&expirationDate)
			up, err := db.Query(
			`UPDATE users SET expiration_date = $1 WHERE
			 id = $2::varchar, activated = $3::bool,
			 email = $4::varchar, first_name = $5::varchar,
			 last_name = $6::varchar, is_admin = $7::bool
			 RETURNING expiration_date
			`,
			expirationDate, id, activated, email, firstName, lastName, isAdmin)
			defer up.Close()

			if !up.Next() {
				return nil, UserNotCreated
			}
			up.Scan(&expirationDate)
	}

	if err != nil {
		switch err.Error() {
		case "pq: duplicate key value violates unique constraint \"users_pkey\"":
			log.Error("user id exists already")
			return nil, UserDuplicated
		case "pq: duplicate key value violates unique constraint \"users_email_key\"":
			log.Error("user email exists already")
			return nil, UserDuplicated
		}
		return nil, err
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, UserNotCreated
	}

	var user User
	rows.Scan(
		&user.Id, &user.Email,
		&user.Activated, &user.FirstName,
		&user.LastName, &user.IsAdmin,
	)
	user.ExpDate = expirationDate

	return &user, err
}

func DeleteUser(id string) error {
	res, err := db.Exec("DELETE FROM users WHERE id = $1::varchar", id)
	if err != nil {
		return err
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return UserNotFound
	}
	return nil
}

func UpdateUserPassword(id string, password string) error {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	res, err := db.Exec(
		`UPDATE users
		SET password = $1::varchar
		WHERE id = $2::varchar`,
		pass, id)
	if err != nil {
		return err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return UserNotFound
	}
	return nil
}

func UpdateUserPrivilege(id string, rank bool) error {
	res, err := db.Exec(
		`UPDATE users
		SET is_admin = $1::boolean
		WHERE id = $2::varchar`,
		rank, id)
	if err != nil {
		return err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return UserNotFound
	}
	return nil
}

func UpdateUserEmail(id string, email string) error {

	res, err := db.Exec(
		`UPDATE users
		SET email = $1::varchar
		WHERE id = $2::varchar`,
		email, id)
	if err != nil {
		return err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return UserNotFound
	}
	return nil
}

func UpdateUserFirstName(id string, firstname string) error {

	res, err := db.Exec(
		`UPDATE users
		SET first_name = $1::varchar
		WHERE id = $2::varchar`,
		firstname, id)
	if err != nil {
		return err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return UserNotFound
	}
	return nil
}

func UpdateUserExpirationDays(id string, expirationDays string) error {

	if expirationDays != "" {
		expirationDays = expirationDays + "days"
		res, err := db.Exec(
			`UPDATE users
			SET expiration_date = current_timestamp + $1
			WHERE id = $2::varchar`,
			expirationDays, id)
		if err != nil {
			return err
		}

		updated, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if updated == 0 {
			return UserNotFound
		}
	} else {
			log.Info("empty string")
		res, err := db.Exec(
			`UPDATE users
			SET expiration_date = NULL
			WHERE id = $1`,
			id)
		if err != nil {
			return err
		}

		updated, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if updated == 0 {
			return UserNotFound
		}
	}
	return nil
}

func UpdateUserLastName(id string, lastname string) error {

	res, err := db.Exec(
		`UPDATE users
		SET last_name = $1::varchar
		WHERE id = $2::varchar`,
		lastname, id)
	if err != nil {
		return err
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return UserNotFound
	}
	return nil
}

func GetUser(id string) (*User, error) {
	rows, err := db.Query(
		`SELECT id, first_name, last_name, email, is_admin,
			activated, extract(epoch from signup_date), expiration_date
		FROM users
		WHERE id = $1::varchar`,
		id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	if rows.Next() {
		var user User
		var timestamp float64

		var tmp sql.NullString

		err = rows.Scan(
			&user.Id,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.IsAdmin,
			&user.Activated,
			&timestamp,
			&tmp,
		)
		if err != nil {
			return nil, err
		}
		user.ExpDate = ""
		if tmp.Valid {
			user.ExpDate = tmp.String
		}
		// javascript time is in millisecond not in second
		user.SignupDate = int(1000 * timestamp)
		return &user, nil
	}
	return nil, nil
}
