# Schemer

[![Main branch protection](https://github.com/trebent/schemer/actions/workflows/main.yaml/badge.svg?branch=main)](https://github.com/trebent/schemer/actions/workflows/main.yaml)
[![Code scanning](https://github.com/trebent/schemer/actions/workflows/code-scanning.yaml/badge.svg?branch=main)](https://github.com/trebent/schemer/actions/workflows/code-scanning.yaml)

Schemer helps you write structured and well validated config files for any application, yay!

* Write a JSON schema
* Write a golang struct to receive the file-based configuration, corresponding to the JSON schema
* Input both to `schemer.Schemer`, call `Parse` voila! Validated configuration for your application

Schemer supports both environment variable injection and path-based lookups through its reference formatting detection:

```bash
# Reference variable, with default
${ref:path.to.field:default}
# Reference variable, without default
${ref:path.to.field}
# Environment variable, with default
${env:variable:default}
# Reference variable, without default
${env:variable}
```
