package heroku

import (
	"fmt"
	"testing"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccHerokuSpaceInboundRule_Basic(t *testing.T) {
	var space heroku.Space
	spaceName := fmt.Sprintf("tftest1-%s", acctest.RandString(10))
	org := getTestingOrgName()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if org == "" {
				t.Skip("HEROKU_ORGANIZATION is not set; skipping test.")
			}
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHerokuSpaceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: "heroku_space_inbound_rule.foobar",
				Config:       testAccCheckHerokuSpaceInboundRuleConfig_basic(spaceName, org),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHerokuSpaceExists("heroku_space.foobar", &space),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_rule.foobar.0", "action", "allow"),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_rule.foobar.0", "source", "8.8.8.8/32"),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_rule.foobar.1", "action", "allow"),
					resource.TestCheckResourceAttr(
						"heroku_space_inbound_rule.foobar.1", "source", "8.8.8.0/24"),
				),
			},
		},
	})
}

func testAccCheckHerokuSpaceInboundRuleConfig_basic(spaceName, orgName string) string {
	return fmt.Sprintf(`
locals {
	sources = ["8.8.8.8/32", "8.8.8.0/24"]
}

resource "heroku_space" "foobar" {
  name         = "%s"
  organization = "%s"
  region       = "virginia"
}

resource "heroku_space_inbound_rule" "foobar" {
	count = "${length(locals.source)}"

  space = "${heroku_space.foobar.name}"
	action = "allow"
	source = "${element(locals.source, count.index)}"
}
`, spaceName, orgName)
}
