package parser

import (
	"fmt"
	"testing"

	"github.com/cmd-tools/aws-commander/cmd"
)

var awsCommandResult = `{
	"Buckets": [
	  {
		"Name": "sample-bucket",
		"CreationDate": "2024-03-26T06:28:38+00:00"
	  },
	  {
		"Name": "sample-bucket1",
		"CreationDate": "2024-03-26T06:28:38+00:01"
	  },
	  {
		"Name": "sample-bucket2",
		"CreationDate": "2024-03-26T06:28:38+00:02"
	  },
	  {
		"Name": "sample-bucket3",
		"CreationDate": "2024-03-26T06:28:38+00:03"
	  }
	],
	"Owner": {
	  "DisplayName": "webfile",
	  "ID": "75aa57f09aa0c8caeab4f8c24e99d10f8e7faeebf76c078efc7c6caea54ba06a"
	}
  }
`

var awsCommandResult2 = `{
	"TableNames": [
	  "global01"
	]
  }  
`

var awsCommandResult3 = `{
	"Items": [
	  {
		"id": {
		  "S": "foo4"
		}
	  },
	  {
		"id": {
		  "S": "foo1"
		}
	  }
  	]
}
`

func Test_ParseCommand_Object(t *testing.T) {
	var commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Buckets",
		},
	}

	var jsonResult1 = ParseCommand(commandTest, awsCommandResult)
	fmt.Println(jsonResult1)

	commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Owner",
		},
	}
	jsonResult1 = ParseCommand(commandTest, awsCommandResult)
	fmt.Println(jsonResult1)

	commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Items",
		},
	}
	jsonResult1 = ParseCommand(commandTest, awsCommandResult3)
	fmt.Println(jsonResult1)
}

func Test_ParseCommand_Array(t *testing.T) {
	var commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "list",
			AttributeName: "TableNames",
		},
	}

	var jsonResult1 = ParseCommand(commandTest, awsCommandResult2)
	fmt.Println(jsonResult1)
}
