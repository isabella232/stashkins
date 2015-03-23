package stashkins

import (
	"testing"

	"github.com/xoom/jenkins"
)

var maven string = `<?xml version='1.0' encoding='UTF-8'?>
<maven2-moduleset plugin="maven-plugin@2.7.1">
    <description>Build maven</description>
    </maven2-moduleset>`

var freestyle string = `<?xml version='1.0' encoding='UTF-8'?>
<project>
  <description>Builds freestyle</description>
</project>`

var unknown string = `<?xml version='1.0' encoding='UTF-8'?>
<nope>
  <description>Builds freestyle</description>
</nope>`

func TestDocumentType(t *testing.T) {
	if jobType, _ := templateType([]byte(maven)); jobType != jenkins.Maven {
		t.Fatalf("Want jenkins.Maven type but got %v\n", jobType)
	}

	if jobType, _ := templateType([]byte(freestyle)); jobType != jenkins.Freestyle {
		t.Fatalf("Want jenkins.Freestyle type but got %v\n", jobType)
	}

	if jobType, _ := templateType([]byte(unknown)); jobType != jenkins.Unknown {
		t.Fatalf("Want jenkins.Unknown type but got %v\n", jobType)
	}

	if jobType, _ := templateType([]byte("")); jobType != jenkins.Unknown {
		t.Fatalf("Want jenkins.Unknown type but got %v\n", jobType)
	}

	if _, err := templateType([]byte("{}")); err == nil {
		t.Fatalf("Expecting an error parsing not-XMLt %v\n", err)
	}
}
