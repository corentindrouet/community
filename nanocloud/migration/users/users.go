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
	"github.com/Nanocloud/community/nanocloud/connectors/db"
	"github.com/Nanocloud/community/nanocloud/models/users"
	"github.com/Nanocloud/community/nanocloud/utils"
)

func createUsersTable() (bool, error) {
	rows, err := db.Query(
		`SELECT table_name
			FROM information_schema.tables
			WHERE table_name = 'users'`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		return false, nil
	}

	rows, err = db.Query(
		`CREATE TABLE users (
			id               varchar(36)                PRIMARY KEY,
			first_name       varchar(36)                NOT NULL DEFAULT '',
			last_name        varchar(36)                NOT NULL DEFAULT '',
			email            varchar(36)                NOT NULL DEFAULT '' UNIQUE,
			password         varchar(60)                NOT NULL DEFAULT '',
			signup_date      timestamp with time zone   NOT NULL DEFAULT current_timestamp,
			expiration_date  timestamp with time zone   DEFAULT NULL,
			is_admin         boolean,
			activated        boolean
		);`)
	if err != nil {
		return false, err
	}

	rows.Close()
	return true, nil
}

func Migrate() error {
	insertAdmin, err := createUsersTable()
	if err != nil {
		return err
	}

	if insertAdmin {
		adminpwd := utils.Env("ADMIN_PASSWORD", "Nanocloud123+")
		adminfirstname := utils.Env("ADMIN_FIRSTNAME", "Admin")
		adminlastname := utils.Env("ADMIN_LASTNAME", "Nanocloud")
		adminmail := utils.Env("ADMIN_MAIL", "admin@nanocloud.com")

		_, err := users.CreateUser(
			true,
			adminmail,
			adminfirstname,
			adminlastname,
			adminpwd,
			true,
			"",
		)

		if err != nil {
			return err
		}
	}

	return nil
}
