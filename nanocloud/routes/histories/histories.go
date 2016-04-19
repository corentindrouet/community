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

package histories

import (
	"net/http"

	"github.com/Nanocloud/community/nanocloud/models/histories"
	"github.com/Nanocloud/community/nanocloud/utils"
	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
)

type hash map[string]interface{}

// Get a list of all the log entries of the database
func List(c *echo.Context) error {

	historyList, err := histories.GetAll()

	if err != nil {
		return err
	}

	var response = make([]hash, len(historyList))
	for i, val := range historyList {
		res := hash{
			"id":         val.Id,
			"type":       "history",
			"attributes": val,
		}
		response[i] = res
	}

	return c.JSON(http.StatusOK, hash{"data": response})

}

// Add a new log entry to the database
func Add(c *echo.Context) error {
	var attr hash

	err := utils.ParseJSONBody(c, &attr)
	if err != nil {
		return err
	}

	data, ok := attr["data"].(map[string]interface{})
	if ok == false {
		return c.JSON(http.StatusBadRequest, hash{
			"error": [1]hash{
				hash{
					"detail": "data is missing",
				},
			},
		})
	}

	attributes, ok := data["attributes"].(map[string]interface{})
	if ok == false {
		return c.JSON(http.StatusBadRequest, hash{
			"error": [1]hash{
				hash{
					"detail": "attributes is missing",
				},
			},
		})
	}

	user_id, ok := attributes["user-id"].(string)
	connection_id, ok := attributes["connection-id"].(string)
	start_date, ok := attributes["start-date"].(string)
	end_date, ok := attributes["end-date"].(string)
	if user_id == "" || connection_id == "" || start_date == "" || end_date == "" {
		log.Error("Missing one or several parameters to create entry")
		return c.JSON(http.StatusBadRequest, hash{
			"error": [1]hash{
				hash{
					"detail": "Missing parameters",
				},
			},
		})
	}

	newHistory, err := histories.CreateHistory(
		user_id,
		connection_id,
		start_date,
		end_date,
	)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, hash{
		"data": hash{
			"id":         newHistory.Id,
			"type":       "history",
			"attributes": newHistory,
		},
	})
}