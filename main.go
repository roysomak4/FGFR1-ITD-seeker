package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
    // FGFR1 gene BED file for variant calling
    bedFile := filepath.Join("bedfiles", *genomeVersion, "FGFR1_gene.bed")
    if err := validateFileExists(bedFile, "BED file"); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s%s FGFR1 gene BED file confirmed%s\n", colorGreen, iconCheck, colorReset)
    fmt.Printf("%s%s Preflight checks passed%s\n", colorGreen, iconCheck, colorReset)
    // ----------------------------------------
    fmt.Printf("%s%s Loading FGFR1 breakpoint exon coordinates...%s\n", colorBlue, iconInfo, colorReset)
    // Load exon coordinates from BED file for validating FGFR1 ITD breakpoints
    exonCoordsBedFile := filepath.Join("bedfiles", *genomeVersion, "FGFR1_ITD_breakpoint_exons.bed")
    if err := validateFileExists(exonCoordsBedFile, "exon coordinates BED file"); err != nil {
        log.Fatal(err)
    }
    exonCoords, err := loadExonCoordinates(exonCoordsBedFile)
    if err != nil {
        log.Fatalf("Failed to load exon coordinates: %v", err)
    }
    fmt.Printf("%s%s Loaded exon coordinates%s\n", colorGreen, iconCheck, colorReset)

    const minSVAltlen = 7000
    const nucleotideExtnLen = 6000
    intermediateVCF := "/tmp/vardict_raw_output.vcf"

    fmt.Printf("%s%s Running vardict command...%s\n", colorCyan, iconSearch, colorReset)
    command := fmt.Sprintf(`~/biotools/vardict -G %s -f %f -r 4 -o 1.5 -th %d -L %d -x %d -N %s -b %s -c 1 -S 2 -E 3 -g 4 %s | ~/biotools/vardict_app/bin/teststrandbias.R | ~/biotools/vardict_app/bin/var2vcf_valid.pl -A -N %s -E -f %f >%s`, *refGenome, *minVaf, *threads, minSVAltlen, nucleotideExtnLen, *sampleName, *inputBam, bedFile, *sampleName, *minVaf, intermediateVCF)
    runBashCommand(command)
    fmt.Printf("%s%s Finished running vardict command%s\n", colorGreen, iconCheck, colorReset)
    
    // Filter VCF for FGFR1 ITD variants
    fmt.Printf("%s%s Filtering VCF for FGFR1 ITD variants...%s\n", colorCyan, iconSearch, colorReset)
    if err := filterVCFForITD(intermediateVCF, *outputVcf, exonCoords); err != nil {
        log.Fatalf("Failed to filter VCF: %v", err)
    }
    fmt.Printf("%s%s Filtered VCF written to %s%s\n", colorGreen, iconCheck, *outputVcf, colorReset)

    // Delete intermediate VCF file
    fmt.Printf("%s%s Deleting intermediate VCF file...%s\n", colorYellow, iconInfo, colorReset)
    if err := os.Remove(intermediateVCF); err != nil {
        log.Printf("%s%s Warning: failed to delete intermediate VCF file: %v%s", colorYellow, iconWarning, err, colorReset)
    }
}

type ExonCoordinates struct {
	prime5Start     int
	prime5End  int
	prime3Start  int
	prime3End    int
}

func loadExonCoordinates(bedFile string) (*ExonCoordinates, error) {
	file, err := os.Open(bedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open exon coordinates BED file: %v", err)
	}
	defer file.Close()

	coords := &ExonCoordinates{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue // Skip comments and empty lines
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue // Skip malformed lines
		}

		exonName := strings.Split(fields[3], ";")[2]
		start, err1 := strconv.Atoi(fields[1])
		end, err2 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil {
			continue
		}

		// Map exon names to coordinates
		switch exonName {
		case "exon-9-10":
			coords.prime5Start = start
			coords.prime5End = end
		case "exon-18":
			coords.prime3Start = start
			coords.prime3End = end
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading breakpoint exon BED file: %v", err)
	}

	// Validate that all required coordinates were found
	if coords.prime5Start == 0 || coords.prime5End == 0 || coords.prime3Start == 0 || coords.prime3End == 0 {
		return nil, fmt.Errorf("BED file is missing required exon coordinates (exon-9-10, exon18)")
	}

	return coords, nil
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
    cmd := exec.Command("bash", "-c", command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Command failed: %v\n%s", err, output)
        return
    }
    fmt.Println(string(output))
}