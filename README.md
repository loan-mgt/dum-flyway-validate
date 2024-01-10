# Dum Flyway Validate

Dum Flyway Validate is a command-line tool to locally validate Flyway migrations. It ensures migration consistency using Git and specific conditions.

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

## Conditions Checked

- **Modified Migration File:** Error if modified after being applied.

- **Added Migration File:** Error if not alphabetically last in the specified directory.

- **Removed Migration File:** Error if removed after being applied.

- **Renamed Migration File:** Error if renamed after being applied.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## License

[GNU3 License](LICENSE)
