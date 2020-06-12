package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/gruntwork-cli/collections"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/gruntwork-cli/shell"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// CreateCli - Create the CLI app with all commands, flags, and usage text configured.
func CreateCli(version string) *cli.App {
	app := cli.NewApp()

	app.Name = "cloud-nuke"
	app.HelpName = app.Name
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.Version = version
	app.Usage = "A CLI tool to nuke (delete) cloud resources."
	app.Commands = []cli.Command{
		{
			Name:   "aws",
			Usage:  "BEWARE: DESTRUCTIVE OPERATION! Nukes AWS resources (ASG, ELB, ELBv2, EBS, EC2, AMI, Snapshots, Elastic IP, RDS).",
			Action: errors.WithPanicHandling(awsNuke),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "region",
					Usage: "Regions to include. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "Regions to exclude. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "resource-type",
					Usage: "Resource types to nuke. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "exclude-resource-type",
					Usage: "Resource types to exclude from nuking. Include multiple times if more than one.",
				},
				cli.StringSliceFlag{
					Name:  "resource-name-pattern",
					Usage: "Regex pattern of resource names to nuke.",
				},
				cli.StringSliceFlag{
					Name:  "exclude-resource-name-pattern",
					Usage: "Regex pattern of resource names to exclude from nuking.",
				},
				cli.StringSliceFlag{
					Name:  "require-resource-tag",
					Usage: "Resource tag to include in nuking. Resource tags should be provided in the form 'tag_name' or 'tag_name:regex_pattern_match'. If the tag_name is provided alone, the tag value is ignored. The tag_name and the regex pattern should be delimited by a single colon. (The tag_name may not also include a colon.)",
				},
				cli.StringSliceFlag{
					Name:  "exclude-resource-tag",
					Usage: "Resource tag to exclude from nuking. Resource tags should be provided in the form 'tag_name' or 'tag_name:regex_pattern_match'. If the tag_name is provided alone, the tag value is ignored. The tag_name and the regex pattern should be delimited by a single colon. (The tag_name may not also include a colon.)",
				},
				cli.BoolFlag{
					Name:  "list-resource-types",
					Usage: "List available resource types",
				},
				cli.StringFlag{
					Name:  "older-than",
					Usage: "Only delete resources older than this specified value. Can be any valid Go duration, such as 10m or 8h.",
					Value: "0s",
				},
				cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Dry run without taking any action.",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Skip nuke confirmation prompt. WARNING: this will automatically delete all resources without any confirmation",
				},
				cli.StringFlag{
					Name:   "log-level",
					Value:  "info",
					Usage:  "Set log level",
					EnvVar: "LOG_LEVEL",
				},
			},
		}, {
			Name:   "defaults-aws",
			Usage:  "Nukes AWS default VPCs and permissive default security group rules. Optionally include/exclude specified regions, or just nuke security group rules (not default VPCs).",
			Action: errors.WithPanicHandling(awsDefaults),
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "region",
					Usage: "regions to include",
				},
				cli.StringSliceFlag{
					Name:  "exclude-region",
					Usage: "regions to exclude",
				},
				cli.BoolFlag{
					Name:  "sg-only",
					Usage: "Destroy default security group rules only. Do not destroy default VPCs.",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "Skip confirmation prompt. WARNING: this will automatically delete defaults without any confirmation",
				},
			},
		},
	}

	return app
}

func parseDurationParam(paramValue string) (*time.Time, error) {
	duration, err := time.ParseDuration(paramValue)
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	// make it negative so it goes back in time
	duration = -1 * duration

	excludeAfter := time.Now().Add(duration)
	return &excludeAfter, nil
}

func awsNuke(c *cli.Context) error {
	logLevel := c.String("log-level")

	parsedLogLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Invalid log level - %s - %s", logLevel, err)
	}
	logging.Logger.Level = parsedLogLevel

	allResourceTypes := aws.ListResourceTypes()

	if c.Bool("list-resource-types") {
		for _, resourceType := range aws.ListResourceTypes() {
			fmt.Println(resourceType)
		}
		return nil
	}

	// Check mutually exclusive filtering options.
	resourceTypes := c.StringSlice("resource-type")
	excludeResourceTypes := c.StringSlice("exclude-resource-type")
	if len(resourceTypes) > 0 && len(excludeResourceTypes) > 0 {
		return fmt.Errorf("You cannot specify both --resource-type and --exclude-resource-type.")
	}
	resourceNamePattern := c.StringSlice("resource-name-pattern")
	excludeResourceNamePattern := c.StringSlice("exclude-resource-name-pattern")
	if len(resourceNamePattern) > 0 && len(excludeResourceNamePattern) > 0 {
		return fmt.Errorf("You cannot specify both --resource-name-pattern and --exclude-resource-name-pattern.")
	}
	requireResourceTag := c.StringSlice("require-resource-tag")
	excludeResourceTag := c.StringSlice("exclude-resource-tag")
	if len(requireResourceTag) > 0 && len(excludeResourceTag) > 0 {
		return fmt.Errorf("You cannot specify both --require-resource-tag and --exclude-resource-tag.")
	}

	// Var check to make sure only allowed resource types are included in the --resource-type or --exclude-resource-type
	// args.
	invalidresourceTypes := []string{}
	for _, resourceType := range resourceTypes {
		if resourceType == "all" {
			continue
		}
		if !aws.IsValidResourceType(resourceType, allResourceTypes) {
			invalidresourceTypes = append(invalidresourceTypes, resourceType)
		}
	}

	for _, resourceType := range excludeResourceTypes {
		if !aws.IsValidResourceType(resourceType, allResourceTypes) {
			invalidresourceTypes = append(invalidresourceTypes, resourceType)
		}
	}

	if len(invalidresourceTypes) > 0 {
		msg := "Try --list-resource-types to get list of valid resource types."
		return fmt.Errorf("Invalid resourceTypes %s specified: %s", invalidresourceTypes, msg)
	}

	// Handle exclude resource types by going through the list of all types and only include those that are not
	// mentioned in the exclude list.
	if len(excludeResourceTypes) > 0 {
		for _, resourceType := range allResourceTypes {
			if !collections.ListContainsElement(excludeResourceTypes, resourceType) {
				resourceTypes = append(resourceTypes, resourceType)
			}
		}
	}

	// Log which resource types will be nuked
	logging.Logger.Info("The following resources types will be nuked:")
	if len(resourceTypes) > 0 {
		for _, resourceType := range resourceTypes {
			logging.Logger.Infof("- %s", resourceType)
		}
	} else {
		for _, resourceType := range allResourceTypes {
			logging.Logger.Infof("- %s", resourceType)
		}
	}

	regions, err := aws.GetEnabledRegions()
	if err != nil {
		return errors.WithStackTrace(err)
	}

	selectedRegions := c.StringSlice("region")
	excludedRegions := c.StringSlice("exclude-region")

	// targetRegions uses selectedRegions and excludedRegions to create a final
	// target region slice.
	targetRegions, err := aws.GetTargetRegions(regions, selectedRegions, excludedRegions)
	if err != nil {
		return fmt.Errorf("Failed to select regions: %s", err)
	}

	excludeAfter, err := parseDurationParam(c.String("older-than"))
	if err != nil {
		return errors.WithStackTrace(err)
	}

	logging.Logger.Infof("Retrieving active AWS resources in [%s]", strings.Join(targetRegions[:], ", "))
	account, err := aws.GetAllResources(
		targetRegions, *excludeAfter, resourceTypes,
		resourceNamePattern, excludeResourceNamePattern,
		requireResourceTag, excludeResourceTag
	)

	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(account.Resources) == 0 {
		logging.Logger.Infoln("Nothing to nuke, you're all good!")
		return nil
	}

	logging.Logger.Infoln("The following AWS resources are going to be nuked: ")

	for region, resourcesInRegion := range account.Resources {
		for _, resources := range resourcesInRegion.Resources {
			for _, identifier := range resources.ResourceIdentifiers() {
				logging.Logger.Infof("* %s-%s-%s\n", resources.ResourceName(), identifier, region)
			}
		}
	}

	if c.Bool("dry-run") {
		logging.Logger.Infoln("Not taking any action as dry-run set to true.")
		return nil
	}

	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke all listed resources? Enter 'nuke' to confirm (or exit with ^C): "
		proceed, err := confirmationPrompt(prompt, 2)
		if err != nil {
			return err
		}
		if proceed {
			if err := aws.NukeAllResources(account, regions); err != nil {
				return err
			}
		}
	} else {
		logging.Logger.Infoln("The --force flag is set, so waiting for 10 seconds before proceeding to nuke everything in your account. If you don't want to proceed, hit CTRL+C now!!")
		for i := 10; i > 0; i-- {
			fmt.Printf("%d...", i)
			time.Sleep(1 * time.Second)
		}

		fmt.Println()
		if err := aws.NukeAllResources(account, regions); err != nil {
			return err
		}
	}

	return nil
}

func awsDefaults(c *cli.Context) error {
	logging.Logger.Infoln("Identifying enabled regions")
	regions, err := aws.GetEnabledRegions()
	if err != nil {
		return errors.WithStackTrace(err)
	}
	for _, region := range regions {
		logging.Logger.Infof("Found enabled region %s", region)
	}

	selectedRegions := c.StringSlice("region")
	excludedRegions := c.StringSlice("exclude-region")

	// targetRegions uses selectedRegions and excludedRegions to create a final
	// target region slice.
	targetRegions, err := aws.GetTargetRegions(regions, selectedRegions, excludedRegions)
	if err != nil {
		return fmt.Errorf("Failed to select regions: %s", err)
	}

	if c.Bool("sg-only") {
		logging.Logger.Info("Not removing default VPCs.")
	} else {
		err = nukeDefaultVpcs(c, targetRegions)
		if err != nil {
			return errors.WithStackTrace(err)
		}
	}

	err = nukeDefaultSecurityGroups(c, targetRegions)
	if err != nil {
		return errors.WithStackTrace(err)
	}
	return nil
}

func nukeDefaultVpcs(c *cli.Context, regions []string) error {
	logging.Logger.Infof("Discovering default VPCs")
	vpcPerRegion := aws.NewVpcPerRegion(regions)
	vpcPerRegion, err := aws.GetDefaultVpcs(vpcPerRegion)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	if len(vpcPerRegion) == 0 {
		logging.Logger.Info("No default VPCs found.")
		return nil
	}

	for _, vpc := range vpcPerRegion {
		logging.Logger.Infof("* Default VPC %s %s", vpc.VpcId, vpc.Region)
	}

	var proceed bool
	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke the default VPCs listed above? Enter 'nuke' to confirm (or exit with ^C): "
		proceed, err = confirmationPrompt(prompt, 2)
		if err != nil {
			return err
		}
	}

	if proceed || c.Bool("force") {
		err := aws.NukeVpcs(vpcPerRegion)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	return nil
}

func nukeDefaultSecurityGroups(c *cli.Context, regions []string) error {
	logging.Logger.Infof("Discovering default security groups")
	defaultSgs, err := aws.GetDefaultSecurityGroups(regions)
	if err != nil {
		return errors.WithStackTrace(err)
	}

	for _, sg := range defaultSgs {
		logging.Logger.Infof("* Default rules for SG %s %s %s", sg.GroupId, sg.GroupName, sg.Region)
	}

	var proceed bool
	if !c.Bool("force") {
		prompt := "\nAre you sure you want to nuke the rules in these default security groups ? Enter 'nuke' to confirm (or exit with ^C): "
		proceed, err = confirmationPrompt(prompt, 2)
		if err != nil {
			return err
		}
	}

	if proceed || c.Bool("force") {
		err := aws.NukeDefaultSecurityGroupRules(defaultSgs)
		if err != nil {
			logging.Logger.Errorf("[Failed] %s", err)
		}
	}
	return nil
}

func confirmationPrompt(prompt string, maxPrompts int) (bool, error) {
	color := color.New(color.FgHiRed, color.Bold)
	color.Println("\nTHE NEXT STEPS ARE DESTRUCTIVE AND COMPLETELY IRREVERSIBLE, PROCEED WITH CAUTION!!!")

	shellOptions := shell.ShellOptions{Logger: logging.Logger}

	// retry prompt on invalid input so user can avoid rescanning all resources
	prompts := 0
	for prompts < maxPrompts {
		input, err := shell.PromptUserForInput(prompt, &shellOptions)

		if err != nil {
			return false, errors.WithStackTrace(err)
		}

		if strings.ToLower(input) == "nuke" {
			return true, nil
		}

		fmt.Printf("Invalid value '%s' was entered.\n", input)
		prompts++
	}

	return false, nil
}
