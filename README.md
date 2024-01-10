<div align="center">
<img width=200 alt="logo Dum Flyway Validate" src="doc/assets/logo.svg">

# Dum Flyway Validate

Dum Flyway Validate is a command-line tool to locally validate Flyway migrations. It ensures migration consistency using Git and specific conditions.

</div>

## Usage

### Basic Usage

```bash
./dum-flyway-validate
```

### Specify Migration Directory

```bash
./dum-flyway-validate --migration-dir path/to/migrations
```

### Enable Debug Mode

```bash
./dum-flyway-validate --migration-dir path/to/migrations --debug
```

### Specify Branch for Comparison

```bash
./dum-flyway-validate --migration-dir path/to/migrations --branch your-branch
```

## Conditions Checked

- **Modified Migration File:** Error if modified after being applied.

- **Added Migration File:** Error if not alphabetically last in the specified directory.

- **Removed Migration File:** Error if removed after being applied.

- **Renamed Migration File:** Error if renamed after being applied.

## Additional Options

- `--migration-dir`: Specifies the migration directory (default: current directory).
- `--branch`: Specifies the branch to compare against (default: empty, i.e., working directory).
- `--debug`: Enable debug mode.


## Continuous Integration Example

Here is an example of integrating Dum Flyway Validate into your CI pipeline:

```yaml
stages:
  - validate

variables:
  DUM_FLYWAY_VALIDATE_VERSION: "v0.2.4"
  MIGRATION_DIR: "path/to/migrations"
  BRANCH_TO_COMPARE: "origin/your-branch"

validate:
  stage: validate
  image: alpine:latest
  script:
    - apk --update add curl git
    - curl -LO https://github.com/Qypol342/dum-flyway-validate/releases/download/$DUM_FLYWAY_VALIDATE_VERSION/dum-flyway-validate
    - chmod +x dum-flyway-validate
    - ./dum-flyway-validate --migration-dir $MIGRATION_DIR --branch $BRANCH_TO_COMPARE
```


## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## License

[GNU3 License](LICENSE)
