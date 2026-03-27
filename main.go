package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Version is set via ldflags at build time
var version = "dev"

// ANSI color codes
const (
    colorReset  = "\033[0m"
    colorRed    = "\033[31m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorBlue   = "\033[94m"  // Bright blue
    colorCyan   = "\033[36m"
    colorPurple = "\033[35m"  // Purple/Magenta
    colorWhite  = "\033[37m"
    
    // Icons
    iconCheck   = "✓"
    iconCross   = "✗"
    iconInfo    = "ℹ"
    iconWarning = "⚠"
    iconSearch  = "🔍"
    iconDNA     = "🧬"
)

func main() {
    // Parse command-line arguments
    refGenome := flag.String("ref", "", "Path to reference genome that was used to generate the BAM file. Must be indexed for the aligner used (e.g. BWA-MEM)")
    inputBam := flag.String("inputBam", "", "Path to sorted, deduped, indexed, and realigned BAM file")
    outputVcf := flag.String("outVcf", "", "Path to output VCF file to write FGFR1 ITD variant calls")
    sampleName := flag.String("sampleName", "", "Sample name for file and VCF header annotations")
    minVaf := flag.Float64("minVaf", 0.01, "Minimum variant allele frequency (VAF) to report a variant (default: 0.01 or 1%). Acceptable value should be a floating point number between 0 and 1.")
    threads := flag.Int("threads", 4, "Number of threads to use (default: 4)")
    genomeVersion := flag.String("genomeVer", "hg38", "Human genome version (e.g., hg19, hg38. Default is hg38) for loading exon coordinates")
	showVersion := flag.Bool("version", false, "Show version information")
    
	flag.Parse()

	// Handle version flag
    if *showVersion {
        fmt.Printf("FGFR1-ITD-seeker v%s\n", version)
        os.Exit(0)
    }

    if *refGenome == "" || *inputBam == "" || *outputVcf == "" || *sampleName == "" {
        flag.Usage()
        log.Fatal("Missing required arguments.")
    }

    // Preflight checks: validate that required files exist
    fmt.Println("")
    fmt.Printf("%s%s Starting variant calling for FGFR1 ITD%s\n", colorCyan, iconDNA, colorReset)
    fmt.Printf("%s%s Genome version: %s%s\n", colorBlue, iconInfo, *genomeVersion, colorReset)
    fmt.Printf("%s%s Performing preflight checks...%s\n", colorYellow, iconSearch, colorReset)

    cfg, ok := genomeConfigs[*genomeVersion]
    if !ok {
        log.Fatalf("Unsupported genome version %q. Supported versions: hg19, hg38", *genomeVersion)
    }

    if err := validateFileExists(*refGenome, "reference genome"); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s%s Reference genome confirmed%s\n", colorGreen, iconCheck, colorReset)
    if err := validateFileExists(*inputBam, "input BAM file"); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s%s Input BAM file confirmed%s\n", colorGreen, iconCheck, colorReset)
    // Validate that BAM index file exists
    bamIndexFile := strings.TrimSuffix(*inputBam, ".bam") + ".bai"
    if err := validateFileExists(bamIndexFile, "BAM index file"); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s%s BAM index file confirmed%s\n", colorGreen, iconCheck, colorReset)
    fmt.Printf("%s%s Preflight checks passed%s\n", colorGreen, iconCheck, colorReset)

    fmt.Printf("%s%s FGFR1 breakpoint exon coordinates loaded (embedded, %s)%s\n", colorGreen, iconCheck, *genomeVersion, colorReset)

    const minSVAltlen = 7000
    const nucleotideExtnLen = 6000
    intermediateVCF := "/tmp/vardict_raw_output.vcf"

    fmt.Printf("%s%s Running vardict command...%s\n", colorCyan, iconSearch, colorReset)
    command := fmt.Sprintf(`vardict -G %s -f %f -r 4 -o 1.5 -th %d -L %d -x %d -N %s -b %s -R %s | teststrandbias.R | var2vcf_valid.pl -A -N %s -E -f %f >%s`, *refGenome, *minVaf, *threads, minSVAltlen, nucleotideExtnLen, *sampleName, *inputBam, cfg.vardictRegion, *sampleName, *minVaf, intermediateVCF)
    runBashCommand(command)
    fmt.Printf("%s%s Finished running vardict command%s\n", colorGreen, iconCheck, colorReset)

    // Filter VCF for FGFR1 ITD variants
    fmt.Printf("%s%s Filtering VCF for FGFR1 ITD variants...%s\n", colorCyan, iconSearch, colorReset)
    if err := filterVCFForITD(intermediateVCF, *outputVcf, &cfg.exonCoords); err != nil {
        log.Fatalf("Failed to filter VCF: %v", err)
    }
    fmt.Printf("%s%s Filtered VCF written to %s%s\n", colorGreen, iconCheck, *outputVcf, colorReset)

    // Delete intermediate VCF file
    fmt.Printf("%s%s Deleting intermediate VCF file...%s\n", colorYellow, iconInfo, colorReset)
    if err := os.Remove(intermediateVCF); err != nil {
        log.Printf("%s%s Warning: failed to delete intermediate VCF file: %v%s", colorYellow, iconWarning, err, colorReset)
    }
}

// ExonCoordinates holds the breakpoint exon boundaries used for ITD filtering.
type ExonCoordinates struct {
	prime5Start int
	prime5End   int
	prime3Start int
	prime3End   int
}

// genomeConfig bundles the VarDict region string with the breakpoint exon coordinates
// for a specific reference genome. Coordinates are sourced from the FGFR1 RefSeq
// transcript NM_023110.3.
type genomeConfig struct {
	vardictRegion string
	exonCoords    ExonCoordinates
}

// genomeConfigs contains the embedded FGFR1 coordinates for each supported genome version.
// Gene region: full FGFR1 gene used as the VarDict target region.
// Exon 18 (prime3): 3' ITD breakpoint region.
// Exon 9-10 (prime5): 5' ITD breakpoint region.
var genomeConfigs = map[string]genomeConfig{
	"hg38": {
		vardictRegion: "chr8:38409143-38470635",
		exonCoords: ExonCoordinates{
			prime5Start: 38418217,
			prime5End:   38419745,
			prime3Start: 38413617,
			prime3End:   38413814,
		},
	},
	"hg19": {
		vardictRegion: "chr8:38266661-38328153",
		exonCoords: ExonCoordinates{
			prime5Start: 38275735,
			prime5End:   38277263,
			prime3Start: 38271135,
			prime3End:   38271332,
		},
	},
}

func validateFileExists(filePath, fileType string) error {
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        return fmt.Errorf("%s does not exist: %s", fileType, filePath)
    } else if err != nil {
        return fmt.Errorf("error checking %s: %v", fileType, err)
    }
    return nil
}

func filterVCFForITD(inputVCF, outputVCF string, exonCoords *ExonCoordinates) error {
	// Open input VCF file
	inFile, err := os.Open(inputVCF)
	if err != nil {
		return fmt.Errorf("failed to open input VCF: %v", err)
	}
	defer inFile.Close()

	// Create output VCF file
	outFile, err := os.Create(outputVCF)
	if err != nil {
		return fmt.Errorf("failed to create output VCF: %v", err)
	}
	defer outFile.Close()

	scanner := bufio.NewScanner(inFile)
	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// Process VCF file line by line
	totalVariantCount := 0
	itdVariantCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Write all header lines
		if strings.HasPrefix(line, "#") {
			writer.WriteString(line + "\n")
			continue
		}

		// Parse variant line
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue // Skip malformed lines
		}

		posStr := fields[1]
		ref := fields[3]
		alt := fields[4]
		totalVariantCount++

		// Parse variant start position
		variantStart, err := strconv.Atoi(posStr)
		if err != nil {
			continue // Skip if position can't be parsed
		}

		// Filter criteria:
		// 1. alt length > ref length AND alt length >= 4000
		// 2. Variant start position is in exon 18
		if len(alt) > len(ref) && len(alt) >= 4000 &&
			variantStart >= exonCoords.prime3Start && variantStart <= exonCoords.prime3End {
			itdVariantCount++
			writer.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading VCF: %v", err)
	}

	fmt.Printf("Total variants processed: %d\n", totalVariantCount)
	fmt.Printf("FGFR1 ITD variants found: %d\n", itdVariantCount)

	return nil
}

func runBashCommand(command string) {
    cmd := exec.Command("sh", "-c", command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Command failed: %v\n%s", err, output)
        return
    }
    fmt.Println(string(output))
}