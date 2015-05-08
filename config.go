package loggerN

/*
Author: chenwenjiang
Email: senchen2013@gmail.com
Create date: 2015-04-05
*/

import (
	"github.com/Unknwon/goconfig"
)

func NewConfig(configFile string) (*goconfig.ConfigFile, error) {
	return goconfig.LoadConfigFile(configFile)
}
