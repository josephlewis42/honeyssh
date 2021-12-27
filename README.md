# Honey SSH

A medium-interaction honeypot in the spirit of
[Kippo](https://www.honeynet.org/projects/old/kippo/).

Features include:

* Interactive shell with 50+ built-in commands
* Download saving (via `scp` and `wget`)
* Session recording and playback
* Custom base file systems
* In-memory interactive file system
* Report generator
* Machine-readable JSON log

## Documentation

Most commands have help if you supply the `--help` flag.

### Running the honeypot

```bash
# Create a new configuration directory and enter it
mkdir honeypot && cd honeypot

# Initialize the configuration
honeyssh init

# Edit the configuration file config.yaml
nano config.yaml

# (Optional) Generate a new public key from a cryptographically secure RNG

# (Optional) Generate a custom file system image from a container
docker pull ubuntu:latest
docker save ubuntu:latest > tmp-image.tar
honeyssh img2fs tmp-image.tar root_fs.tar.gz

# Test your configuration using the playground functionality
honeyssh playground

# Start the honeypot
honeyssh serve
```

### Configuration

The current directory is used for configuration by default, but can be
overridden by the `--config` flag.

The configuration directory has the following items:

* `app.log`: SSH server event log newline delimited JSON events described by
  `core/logger/log.proto`.
* `config.yaml`: honeypot configuration, see the contents for descriptions of
  each item.
* `downloads`: items downloaded or uploaded by attackers to the honeypot, also
  includes metadata files about the invocation that caused the file to be placed
  here.
* `private_key`: private key the SSH server uses.
* `root_fs.tar.gz`: the root file system, by default this is adapted from
  `gcr.io/distroless`.
* `session_logs`: interactive session log recordings.

### Viewing the logs

### Generating interaction reports

## Is it safe?

Maybe. As a medium interaction honeypot, it's more dangerous than a firewall
that denies all connections, but far safer than giving them access to a
machine/container that you hope you've plugged all the holes in.

Consider running `honeyssh` in [gVisor](https://github.com/google/gvisor) just in
case.

## Contributions

See CONTRIBUTING.md.

## License

`honeyssh` is licensed under the Apache 2 license, see LICENSE for the full text.

Additional licenses can be found in the `third_party/` and `vendor/`
directories.
