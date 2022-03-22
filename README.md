# ssh-cert extension for gh

This [gh](https://cli.github.com) extension allows you to request [a SSH certificate](https://github.blog/2019-08-14-ssh-certificate-authentication-for-github-enterprise-cloud/) to access a GitHub repo leveraging [the SSH certificate authority feature](https://docs.github.com/en/organizations/managing-git-access-to-your-organizations-repositories/about-ssh-certificate-authorities). 

This extension requires the deployment of [the SSH Cert App](https://github.com/helaili/ssh-cert-app) companion GitHub App.

## Installation

1. Install the gh cli - see [the installation/upgrade instructions](https://github.com/cli/cli#installation)

1. Install this extension:

```cmd
gh extension install helaili/ssh-cert-app
```

## Usage
In order to retrieve a certificate, [the SSH Cert App](https://github.com/helaili/ssh-cert-app) must be installed on at least one repository of an organisation. Users requesting a certificate must be authenticated and have write access to this repository. 

```cmd 
gh ssh-cert get
```

When no flags are supplied, we look for parameters in `./.gh-ssh-cert.yaml` and then `$HOME/.gh-ssh-cert.yaml`. 

Sample `.gh-ssh-cert.yaml` file:
```yaml
org: my-org
repo: a-repo
pubKey: '~/.ssh/id_rsa.pub'
url: https://somewhere.com/ssh-cert-app
```

or 

```cmd 
gh ssh-cert get <flags>
```

where flags are: 

`--org` or `-o` -  The GitHub Organization where [the SSH Cert App](https://github.com/helaili/ssh-cert-app) is installed.

`--repo` or `-r` - The repository where [the SSH Cert App](https://github.com/helaili/ssh-cert-app) is installed.

`--pubKey` or `-k` - The Public key file to request a certificate for, e.g. `~/.ssh/id_rsa.pub`. This key needs to exist on your GitHub profile.

`--url` or `-u` - The root URL of your instance of [the SSH Cert App](https://github.com/helaili/ssh-cert-app)

`--config` or `-c` - The YAML config file which will provide the above parameters, in case this file is not `./.gh-ssh-cert.yaml` or `$HOME/.gh-ssh-cert.yaml` 

Note that the `org` and `repo` parameters can be omitted when the command is run from a clone of the GitHub repo, `my-org/a-repo` in our exemple. 

## Develop

- run `git clone https://github.com/helaili/gh-ssh-cert` 
- run `cd gh-ssh-cert; gh extension install .; gh ssh-cert <flags>`
- use `go build && gh ssh-cert <flags>`to see changes in your code as you develop


