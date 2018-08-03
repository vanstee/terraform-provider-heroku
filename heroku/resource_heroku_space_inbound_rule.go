package heroku

import (
	"context"
	"fmt"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

type Rule struct {
	Action string `json:"action" url:"action,key"`
	Source string `json:"source" url:"source,key"`
}

func resourceHerokuSpaceInboundRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceInboundRuleCreate,
		Read:   resourceHerokuSpaceInboundRuleRead,
		Delete: resourceHerokuSpaceInboundRuleDelete,

		Schema: map[string]*schema.Schema{
			"space": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"action": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"allow",
					"deny",
				}, false),
			},
			"source": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCIDRNetworkAddress,
			},
		},
	}
}

func resourceHerokuSpaceInboundRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)
	action := d.Get("action").(string)
	source := d.Get("source").(string)

	herokuMutexKV.Lock(spaceIdentity)
	defer herokuMutexKV.Unlock(spaceIdentity)

	fmt.Printf("[DEBUG] Retrieving current inbound ruleset for space (%s)", spaceIdentity)
	ruleset, err := client.InboundRulesetCurrent(context.Background(), spaceIdentity)
	if err != nil {
		return fmt.Errorf("Error creating inbound rule (%s %s) for space (%s): %s", allow, source, spaceIdentity, err)
	}

	for _, r := range ruleset.Rules {
		if r.Action == action && r.Source == source {
			fmt.Printf("[DEBUG] Rule (%s %s) already added, do nothing", action, source)
			return nil
		}
	}

	rules := make([]*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}, len(ruleset.Rules)+1)
	for i, r := range ruleset.Rules {
		rules[i] = &r
	}

	rules = append(rules, &struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}{
		Action: action,
		Source: source,
	})

	fmt.Printf("[DEBUG] Updating current inbound ruleset for space (%s)", spaceIdentity)
	opts := heroku.InboundRulesetCreateOpts{Rules: rules}
	_, err = client.InboundRulesetCreate(context.Background(), spaceIdentity, opts)
	if err != nil {
		return fmt.Errorf("Error creating inbound rule (%s %s) for space (%s): %s", allow, source, spaceIdentity, err)
	}

	return resourceHerokuSpaceInboundRuleRead(d, meta)
}

func resourceHerokuSpaceInboundRuleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)
	action := d.Get("action").(string)
	source := d.Get("source").(string)

	fmt.Printf("[DEBUG] Retrieving current inbound ruleset for space (%s)", spaceIdentity)
	ruleset, err := client.InboundRulesetCurrent(context.Background(), spaceIdentity)
	if err != nil {
		return fmt.Errorf("Error reading inbound rule (%s %s) for space (%s): %s", allow, source, spaceIdentity, err)
	}

	for _, r := range ruleset.Rules {
		if r.Action == action && r.Source == source {
			d.SetId(r.Source + " " + r.Action)
			d.Set("space", ruleset.Space.Name)
			d.Set("action", r.Action)
			d.Set("source", r.Source)

			return nil
		}
	}

	message := fmt.Sprintf("Rule (%s %s) not found in current inbound ruleset", action, source)
	return fmt.Errorf("Error reading inbound rule for space (%s): %s", spaceIdentity, message)
}

func resourceHerokuSpaceInboundRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)
	action := d.Get("action").(string)
	source := d.Get("source").(string)

	herokuMutexKV.Lock(spaceIdentity)
	defer herokuMutexKV.Unlock(spaceIdentity)

	fmt.Printf("[DEBUG] Retrieving current inbound ruleset for space (%s)", spaceIdentity)
	ruleset, err := client.InboundRulesetCurrent(context.Background(), spaceIdentity)
	if err != nil {
		return fmt.Errorf("Error deleting inbound rule (%s %s) for space (%s): %s", allow, source, spaceIdentity, err)
	}

	index := -1
	for i, r := range ruleset.Rules {
		if r.Action == action && r.Source == source {
			index = i
			break
		}
	}

	if index == -1 {
		fmt.Printf("[DEBUG] Rule (%s %s) already deleted, do nothing", action, source)
		return nil
	}

	rules := make([]*struct {
		Action string `json:"action" url:"action,key"`
		Source string `json:"source" url:"source,key"`
	}, len(ruleset.Rules))
	for i, r := range ruleset.Rules {
		rules[i] = &r
	}

	rules = append(rules[:index], rules[index+1:]...)

	// Heroku Private Spaces ship with a default 0.0.0.0/0 inbound ruleset. An HPS *MUST* have
	// an inbound ruleset. There's no delete API method for this. So when we "delete" the ruleset
	// we reset things back to what Heroku sets when the HPS is created. Given that the default
	// allows all traffic from all places, this is akin to deleting all filtering.
	if len(rules) == 0 {
		fmt.Printf("[DEBUG] Resetting ruleset to allow all inbound traffic (allow 0.0.0.0/0)")
		rules = append(rules, &struct {
			Action string `json:"action" url:"action,key"`
			Source string `json:"source" url:"source,key"`
		}{
			Source: "0.0.0.0/0",
			Action: "allow",
		})
	}

	fmt.Printf("[DEBUG] Updating current inbound ruleset for space (%s)", spaceIdentity)
	opts := heroku.InboundRulesetCreateOpts{Rules: rules}
	_, err = client.InboundRulesetCreate(context.Background(), spaceIdentity, opts)
	if err != nil {
		return fmt.Errorf("Error removing rule (%s %s) for space (%s): %s", allow, source, spaceIdentity, err)
	}

	d.SetId("")
	return nil
}
