package configuration

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/zvonler/espy/database"
	"github.com/zvonler/espy/utils"
)

func OpenExistingDatabase() (sdb *database.ScraperDB, err error) {
	dbPath := viper.GetString("database")

	var exists bool
	if exists, err = utils.PathExists(dbPath); err == nil {
		if exists {
			sdb, err = database.OpenScraperDB(dbPath)
		} else {
			err = fmt.Errorf("Database %q does not exist", dbPath)
		}
	}
	return
}
