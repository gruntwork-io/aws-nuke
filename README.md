[![Maintained by Gruntwork.io](https://img.shields.io/badge/maintained%20by-gruntwork.io-%235849a6.svg)](https://gruntwork.io/?ref=repo_cloud_nuke)

# cloud-nuke

This repo contains a CLI tool to delete all resources in an AWS account. cloud-nuke was created for situations when you might have an account you use for testing and need to clean up leftover resources so you're not charged for them. Also great for cleaning out accounts with redundant resources. Also great for removing unnecessary defaults like default VPCs and permissive ingress/egress rules in default security groups.

The currently supported functionality includes:

## AWS

- Deleting all Auto scaling groups in an AWS account
- Deleting all Elastic Load Balancers (Classic and V2) in an AWS account
- Deleting all Transit Gateways in an AWS account
- Deleting all EBS Volumes in an AWS account
- Deleting all unprotected EC2 instances in an AWS account
- Deleting all AMIs in an AWS account
- Deleting all Snapshots in an AWS account
- Deleting all Elastic IPs in an AWS account
- Deleting all Launch Configurations in an AWS account
- Deleting all ECS services in an AWS account
- Deleting all ECS clusters in an AWS account
- Deleting all EKS clusters in an AWS account
- Deleting all RDS DB instances in an AWS account
- Deleting all Lambda Functions in an AWS account
- Deleting all SQS queues in an AWS account
- Deleting all S3 buckets in an AWS account - except for buckets tagged with Key=cloud-nuke-excluded Value=true
- Deleting all default VPCs in an AWS account
- Deleting all IAM users in an AWS account
- Deleting all Secrets Manager Secrets in an AWS account
- Deleting all NAT Gateways in an AWS account
- Deleting all GuardDuty instances in an AWS account  
- Revoking the default rules in the un-deletable default security group of a VPC

### BEWARE!

When executed as `cloud-nuke aws`, this tool is **HIGHLY DESTRUCTIVE** and deletes all resources! This mode should never be used in a production environment!

When executed as `cloud-nuke defaults-aws`, this tool deletes all DEFAULT VPCs and the default ingress/egress rule for all default security groups. This should be used in production environments **WITH CAUTION**.

## Install

### Download from releases page

1. Download the latest binary for your OS on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases).
2. Move the binary to a folder on your `PATH`. E.g.: `mv cloud-nuke_darwin_amd64 /usr/local/bin/cloud-nuke`.
3. Add execute permissions to the binary. E.g.: `chmod u+x /usr/local/bin/cloud-nuke`.
4. Test it installed correctly: `cloud-nuke --help`.

### Install via package manager

Note that package managers are third party. The third party cloud-nuke packages may not be updated with the latest version, but are often close. Please check your version against the latest available on the [releases page](https://github.com/gruntwork-io/cloud-nuke/releases). If you want the latest version, the recommended installation option is to [download from the releases page](https://github.com/gruntwork-io/cloud-nuke/releases).

- **macOS:** You can install cloud-nuke using [Homebrew](https://brew.sh/): `brew install cloud-nuke`.

- **Linux:** Most Linux users can use [Homebrew](https://docs.brew.sh/Homebrew-on-Linux): `brew install cloud-nuke`.

## Usage

Simply running `cloud-nuke aws` will start the process of cleaning up your cloud account. You'll be shown a list of resources that'll be deleted as well as a prompt to confirm before any deletion actually takes place.

In AWS, to delete only the default resources, run `cloud-nuke defaults-aws`. This will remove the default VPCs in each region, and will also revoke the ingress and egress rules associated with the default security group in each VPC. Note that the default security group itself is unable to be deleted.

### Nuke resources in certain regions

When using `cloud-nuke aws`, you can use the `--region` flag to target resources in certain regions for deletion. For example the following command will nuke resources only in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --region ap-south-1 --region ap-south-2
```

Including regions is available within both `cloud-nuke aws` and with `cloud-nuke defaults-aws`.

### Exclude resources in certain regions

When using `cloud-nuke aws`, you can use the `--exclude-region` flag to exclude resources in certain regions from being deleted. For example the following command does not nuke resources in `ap-south-1` and `ap-south-2` regions:

```shell
cloud-nuke aws --exclude-region ap-south-1 --exclude-region ap-south-2
```

`--region` and `--exclude-region` flags cannot be specified together i.e. they are mutually exclusive.

Excluding regions is available within both `cloud-nuke aws` and with `cloud-nuke defaults-aws`.

### Excluding Resources by Age

You can use the `--older-than` flag to only nuke resources that were created before a certain period, the possible values are all valid values for [ParseDuration](https://golang.org/pkg/time/#ParseDuration) For example the following command nukes resources that are at least one day old:

```shell
cloud-nuke aws --older-than 24h
```

### List supported resource types

You can use the `--list-resource-types` flag to list resource types whose termination is currently supported:

```shell
cloud-nuke aws --list-resource-types
```

### Terminate specific resource types

If you want to target specific resource types (e.g ec2, ami, etc.) instead of all the supported resources you can
do so by specifying them through the `--resource-type` flag:

```shell
cloud-nuke aws --resource-type ec2 --resource-type ami
```

will search and target only `ec2` and `ami` resources. The specified resource type should be a valid resource type
i.e. it should be present in the `--list-resource-types` output. Using `--resource-type` also speeds up search because
we are searching only for specific resource types.

### Exclude terminating specific resource types

Just like you can select which resources to terminate using `--resource-type`, you can select which resources to skip using
`--exclude-resource-type` flag:

```shell
cloud-nuke aws --exclude-resource-type s3 --exclude-resource-type ec2
```

This will terminate all resource types other than S3 and EC2.

`--resource-type` and `--exclude-resource-type` flags cannot be specified together i.e. they are mutually exclusive.

### Dry run mode

If you want to check what resources are going to be targeted without actually terminating them, you can use the
`--dry-run` flag

```shell
cloud-nuke aws --resource-type ec2 --dry-run
```

### Config file

For more granularity, such as being able to specify which resources to terminate using regular expressions or plain text, you can pass in a configuration file.

_Note: Config file support is a new feature and only filtering a handful of resources by name using regular expressions is currently supported. We'll be adding more support in the future, and pull requests are welcome!_

The following resources support the Config file:

- S3 Buckets
    - Resource type: `s3`
    - Config key: `s3`
- IAM Users
    - Resource type: `iam`
    - Config key: `IAMUsers`
- Secrets Manager Secrets
    - Resource type: `secretsmanager`
    - Config key: `SecretsManager`
- NAT Gateways
    - Resource type: `nat-gateway`
    - Config key: `NATGateway`


#### Example

```shell
cloud-nuke aws --resource-type s3 --config path/to/file.yaml
```

Given this command, `cloud-nuke` will nuke _only_ S3 buckets, as specified by the `--resource-type s3` option.

Now given the following config, the s3 buckets that will be nuked are further filtered to only include ones that match any of the provided regular expressions. So a bucket named `alb-app-access-logs` would be deleted, but a bucket named `my-s3-bucket` would not.
```yaml
s3:
  include:
    names_regex:
      - ^alb-.*-access-logs$
      - .*-prod-alb-.*
```

Similarly, you can adjust the config to delete only IAM users of a particular name by using the `IAMUsers` key. For
example, in the following config, only IAM users that have the prefix `my-test-user-` in their username will be deleted.

```
IAMUsers:
  include:
    names_regex:
      - ^my-test-user-.*
```

#### Include and exclude together

Now consider the following contrived example:

```yaml
s3:
  include:
    names_regex:
      - ^alb-.*-access-logs$
      - .*-prod-alb-.*
  exclude:
    names_regex:
      - public
      - prod
```

The intention is to delete all the s3 buckets that match the include rules but not the exclude rules. Filtering is commutative, meaning that you should get the same result whether you apply the include filters before or after the exclude filters.

The result of these filters applied in either order will be a set of s3 buckets that match `^alb-.*-access-logs$` as long as they do not also contain `public` or `prod`. The rule to include s3 buckets matching `.*-prod-alb-.*` is negated by the rule to exclude those matching `prod`.

<!-- We might only want to support region and resource-type in the command line, rather than in the config file.

Given this config, `cloud-nuke` will nuke all S3 buckets that exist in `us-east-1` and all S3 buckets that exist in `us-west-1`.
```yaml
s3:
  include:
    regions:
      - us-east-1
      - us-west-1
```

Given this config, `cloud-nuke` will nuke all S3 buckets that match the regular expression but only if they do not also exist in `us-east-1`. So a bucket named `abc-prod-alb-def` located in the `ap-northeast-2` region would be nuked.
```yaml
s3:
  include:
    names_regex:
      - .*-prod-alb-.*
  exclude:
    regions:
      - us-east-1
```
-->


#### CLI options override config file options

The options provided in the command line take precedence over those provided in any config file that gets passed in. For example, say you provide `--resource-type s3` in the command line, along with a config file that specifies `ec2:` at the top level but doesn't specify `s3:`. The command line argument filters the resource types to include only s3, so the rules in the config file for `ec2:` are ignored, and ec2 resources are not nuked. All s3 resources would be nuked.

In the same vein, say you do not provide a `--resource-type` option in the command line, but you do pass in a config file that only lists rules for `s3:`, such as `cloud-nuke aws --config path/to/config.yaml`. In this case _all_ resources would be nuked, but among `s3` buckets, only those matching your config file rules would be nuked.

Be careful when nuking and append the `--dry-run` option if you're unsure. Even without `--dry-run`, `cloud-nuke` will list resources that would undergo nuking and wait for your confirmation before carrying it out.

#### What's supported?

To find out what we options are supported in the config file today, consult this table. Resource types at the top level of the file that are supported are listed here.

| resource type      | names | names_regex | tags | tags_regex |
|--------------------|-------|-------------|------|------------|
| s3                 | none  | ✅          | none | none       |
| iam                | none  | ✅          | none | none       |
| secretsmanager     | none  | ✅          | none | none       |
| nat-gateway        | none  | ✅          | none | none       |
| ec2 instance       | none  | none        | none | none       |
| iam role           | none  | none        | none | none       |
| ... (more to come) | none  | none        | none | none       |



### Log level

You can set the log level by specifying the `--log-level` flag as per [logrus](https://github.com/sirupsen/logrus) log levels:

```shell
cloud-nuke aws --log-level debug
```

OR

```shell
LOG_LEVEL=debug cloud-nuke aws
```

Default value is - `info`. Acceptable values are `debug, info, warn, error, panic, fatal, trace` as per [logrus log level parser](https://github.com/sirupsen/logrus/blob/master/logrus.go#L25).

### Nuking only default security group rules
When deleting defaults with `cloud-nuke defaults-aws`, use the `--sg-only` flag to delete only the default
security group rules and not the default VPCs.

```shell
cloud-nuke defaults-aws --sg-only
```

Happy Nuking!!!

## Credentials

### AWS

In order for the `cloud-nuke` CLI tool to access your AWS, you will need to provide your AWS credentials. You can use one of the [standard AWS CLI credential mechanisms](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

## Running Tests

```shell
go test -v ./...
```

## Contributing

cloud-nuke is an open source project, and contributions from the community are very welcome! Please check out the
[Contribution Guidelines](CONTRIBUTING.md) and [Developing cloud-nuke](#developing-cloud-nuke) for instructions.

## Developing cloud-nuke

### Running Locally

To run cloud-nuke locally, use the `go run` command:

```bash
go run main.go
```

### Running tests

**Note**: Many of the tests in the `aws` folder run against a real AWS account and will create and destroy actual resources. DO NOT
hit `CTRL+C` while the tests are running, as this will prevent them from cleaning up properly. We are not responsible for any
charges you may incur.

Before running the tests, you must configure your [AWS credentials](#credentials).

To run all the tests:

```bash
go test -v ./...
```

To run only the tests in a specific package, such as the package `aws`:

```bash
cd aws
go test -v
```

And to run a specific test, such as `TestListAMIs` in package `aws`:

```bash
cd aws
go test -v -run TestListAMIs
```

### Formatting

Every source file in this project should be formatted with `go fmt`.

### Releasing new versions

To release a new version, just go to the [Releases Page](https://github.com/gruntwork-io/cloud-nuke/releases) and
create a new release. The CircleCI job for this repo has been configured to:

1. Automatically detect new tags.
1. Build binaries for every OS using that tag as a version number.
1. Upload the binaries to the release in GitHub.

See `.circleci/config.yml` for details.

## License

This code is released under the MIT License. See [LICENSE.txt](/LICENSE.txt).
