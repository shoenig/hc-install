package releases

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"gophers.dev/cmds/hc-install/internal/testutil"
	"gophers.dev/cmds/hc-install/product"
	"gophers.dev/cmds/hc-install/src"
)

func TestVersions_List(t *testing.T) {
	testutil.EndToEndTest(t)

	cons, err := version.NewConstraint(">= 1.0.0, < 1.0.10")
	if err != nil {
		t.Fatal(err)
	}

	versions := &Versions{
		Product:     product.Terraform,
		Constraints: cons,
	}

	ctx := context.Background()
	sources, err := versions.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersions := []string{
		"1.0.0",
		"1.0.1",
		"1.0.2",
		"1.0.3",
		"1.0.4",
		"1.0.5",
		"1.0.6",
		"1.0.7",
		"1.0.8",
		"1.0.9",
	}
	if diff := cmp.Diff(expectedVersions, sourcesToRawVersions(sources)); diff != "" {
		t.Fatalf("unexpected versions: %s", diff)
	}
}

func sourcesToRawVersions(srcs []src.Source) []string {
	rawVersions := make([]string, len(srcs))

	for idx, src := range srcs {
		source := src.(*ExactVersion)
		rawVersions[idx] = source.Version.String()
	}

	return rawVersions
}
