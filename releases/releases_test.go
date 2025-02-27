package releases

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	"gophers.dev/cmds/hc-install/internal/testutil"
	"gophers.dev/cmds/hc-install/product"
	"gophers.dev/cmds/hc-install/src"
)

var (
	_ src.Installable = &ExactVersion{}
	_ src.Removable   = &ExactVersion{}

	_ src.Installable = &LatestVersion{}
	_ src.Removable   = &LatestVersion{}
)

func TestLatestVersion(t *testing.T) {
	testutil.EndToEndTest(t)

	lv := &LatestVersion{
		Product: product.Terraform,
	}
	lv.SetLogger(testutil.TestLogger())

	ctx := context.Background()

	execPath, err := lv.Install(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lv.Remove(ctx) })

	v, err := product.Terraform.GetVersion(ctx, execPath)
	if err != nil {
		t.Fatal(err)
	}

	latestConstraint, err := version.NewConstraint(">= 1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !latestConstraint.Check(v) {
		t.Fatalf("versions don't match (expected: %s, installed: %s)",
			latestConstraint, v)
	}
}

func TestLatestVersion_basic(t *testing.T) {
	mockApiRoot := filepath.Join("testdata", "mock_api_tf_0_14_with_prereleases")
	lv := &LatestVersion{
		Product:          product.Terraform,
		ArmoredPublicKey: getTestPubKey(t),
		apiBaseURL:       testutil.NewTestServer(t, mockApiRoot).URL,
	}
	lv.SetLogger(testutil.TestLogger())

	ctx := context.Background()

	execPath, err := lv.Install(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lv.Remove(ctx) })

	v, err := product.Terraform.GetVersion(ctx, execPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersion, err := version.NewVersion("0.14.11")
	if err != nil {
		t.Fatal(err)
	}
	if !expectedVersion.Equal(v) {
		t.Fatalf("versions don't match (expected: %s, installed: %s)",
			expectedVersion, v)
	}
}

func TestLatestVersion_prereleases(t *testing.T) {
	mockApiRoot := filepath.Join("testdata", "mock_api_tf_0_14_with_prereleases")

	lv := &LatestVersion{
		Product:            product.Terraform,
		IncludePrereleases: true,
		ArmoredPublicKey:   getTestPubKey(t),
		apiBaseURL:         testutil.NewTestServer(t, mockApiRoot).URL,
	}
	lv.SetLogger(testutil.TestLogger())

	ctx := context.Background()

	execPath, err := lv.Install(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lv.Remove(ctx) })

	v, err := product.Terraform.GetVersion(ctx, execPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersion, err := version.NewVersion("0.15.0-rc2")
	if err != nil {
		t.Fatal(err)
	}
	if !expectedVersion.Equal(v) {
		t.Fatalf("versions don't match (expected: %s, installed: %s)",
			expectedVersion, v)
	}
}

func TestExactVersion(t *testing.T) {
	testutil.EndToEndTest(t)

	versionToInstall := version.Must(version.NewVersion("0.14.0"))
	ev := &ExactVersion{
		Product: product.Terraform,
		Version: versionToInstall,
	}
	ev.SetLogger(testutil.TestLogger())

	ctx := context.Background()

	execPath, err := ev.Install(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { ev.Remove(ctx) })

	t.Logf("exec path of installed: %s", execPath)

	v, err := product.Terraform.GetVersion(ctx, execPath)
	if err != nil {
		t.Fatal(err)
	}

	if !versionToInstall.Equal(v) {
		t.Fatalf("versions don't match (expected: %s, installed: %s)",
			versionToInstall, v)
	}
}

func getTestPubKey(t *testing.T) string {
	f, err := os.Open(filepath.Join("testdata", "2FCA0A85.pub"))
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
