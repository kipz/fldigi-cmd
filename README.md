# fldigi-cmd

A Go CLI tool that polls the fldigi XML-RPC interface to monitor band changes and triggers external programs when the band switches.

## Features

- Monitors fldigi frequency via XML-RPC
- Detects band changes across all amateur radio bands (HF, VHF, UHF, microwave)
- Runs external commands with actual band names when changes occur
- Configurable polling interval and connection settings

## Supported Bands

The tool maps frequencies to actual amateur radio band names:

**HF Bands:**
- `160m`: 1.800 - 2.000 MHz
- `80m`: 3.500 - 4.000 MHz
- `60m`: 5.3305 - 5.4035 MHz
- `40m`: 7.000 - 7.300 MHz
- `30m`: 10.100 - 10.150 MHz
- `20m`: 14.000 - 14.350 MHz
- `17m`: 18.068 - 18.168 MHz
- `15m`: 21.000 - 21.450 MHz
- `12m`: 24.890 - 24.990 MHz
- `10m`: 28.000 - 29.700 MHz

**VHF/UHF Bands:**
- `6m`: 50.000 - 54.000 MHz
- `2m`: 144.000 - 148.000 MHz
- `1.25m`: 222.000 - 225.000 MHz
- `70cm`: 420.000 - 450.000 MHz

**Microwave Bands:**
- `33cm`: 902.000 - 928.000 MHz
- `23cm`: 1240.000 - 1300.000 MHz
- `13cm`: 2300.000 - 2450.000 MHz
- `9cm`: 3300.000 - 3500.000 MHz
- `5cm`: 5650.000 - 5925.000 MHz
- `3cm`: 10000.000 - 10500.000 MHz
- `1.2cm`: 24000.000 - 24250.000 MHz

**LF Bands:**
- `630m`: 472 - 479 kHz
- `2200m`: 135.7 - 137.8 kHz

Frequencies outside these amateur radio bands are ignored.

## Usage

```bash
./fldigi-cmd --command "/path/to/your/script" [options]
```

### Options

- `--command`, `-c string`: External command to run on band change (required)
- `--host`, `-h string`: fldigi host (default "127.0.0.1")
- `--port`, `-p int`: fldigi XML-RPC port (default 7362)
- `--interval`, `-i duration`: polling interval (default 5s)

### Examples

```bash
# Basic usage (long form)
./fldigi-cmd --command "./band-change-handler.sh"

# Basic usage (short form)
./fldigi-cmd -c "./band-change-handler.sh"

# Custom polling interval
./fldigi-cmd --command "echo" --interval 2s

# Remote fldigi instance (mixed short/long)
./fldigi-cmd -c "./handler.sh" --host 192.168.1.100 -p 7362
```

## Requirements

- fldigi or flrig running with XML-RPC enabled
- fldigi/flrig configured to listen on the specified host/port (default: 127.0.0.1:7362)

## Building

```bash
go build -o fldigi-cmd .
```

## External Command

The external command receives the amateur radio band name as the first argument. Examples include:
- `10m`, `12m`, `15m`, `17m`, `20m`, `30m`, `40m`, `60m`, `80m`, `160m`
- `6m`, `2m`, `1.25m`, `70cm`, `33cm`, `23cm`
- `630m`, `2200m`

Example handler script:
```bash
#!/bin/bash
BAND="$1"
echo "Band changed to: $BAND"

case "$BAND" in
    "10m"|"15m"|"20m")
        echo "Switching to HF DX configuration"
        ;;
    "40m"|"80m"|"160m")
        echo "Switching to HF low band configuration"
        ;;
    "6m"|"2m")
        echo "Switching to VHF configuration"
        ;;
    "70cm")
        echo "Switching to UHF configuration"
        ;;
    *)
        echo "Unknown band: $BAND"
        ;;
esac
```