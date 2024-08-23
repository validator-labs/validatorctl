package validator

import (
	"github.com/validator-labs/validator/pkg/validationrule"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// initRule initializes a rule by ensuring its name is set. Optionally prints a message when the
// rule is being reconfigured.
func initRule(rule validationrule.Interface, ruleType, message string, ruleNames *[]string) error {
	name := rule.Name()

	// If it already has a name, we are reconfiguring it. Tell the user and then move on.
	if rule.Name() != "" {
		log.InfoCLI("\nReconfiguring %s rule: %s", ruleType, name)
		if message != "" {
			log.InfoCLI(message)
		}
		*ruleNames = append(*ruleNames, name)

		return nil
	}

	// If it doesn't have a name, we aren't reconfiguring it. Prompt the user for a name.
	if rule.RequiresName() {
		// This also
		name, err := getRuleName(ruleNames)
		if err != nil {
			return err
		}
		rule.SetName(name)
	}

	return nil
}
