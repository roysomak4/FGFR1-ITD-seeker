package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
)

func main() {
    // Parse command-line arguments
    refGenome := flag.String("ref", "", "Path to reference genome that was used to generate the BAM file. Must be indexed for the aligner used (e.g. BWA-MEM)")
    inputBam := flag.String("inputBam", "", "Path to sorted, deduped, indexed, and realigned BAM file")
    bedFile := flag.String("fgfrBed", "", "Path to BED file containing the FGFR1 gene region. The BED file genome version used must correspond to the genome version of the reference genome and BAM file.")
    outputVcf := flag.String("outVcf", "", "Path to output VCF file to write FGFR1 ITD variant calls")
    sampleName := flag.String("sampleName", "", "Sample name for file and VCF header annotations")
    minVaf := flag.Float64("minVaf", 0.01, "Minimum variant allele frequency (VAF) to report a variant (default: 0.01 or 1%). Acceptable value should be a floating point number between 0 and 1.")
    threads := flag.Int("threads", 4, "Number of threads to use (default: 4)")
    flag.Parse()

    if *refGenome == "" || *inputBam == "" || *bedFile == "" || *outputVcf == "" || *sampleName == "" {
        flag.Usage()
        log.Fatal("Missing required arguments.")
    }
	fmt.Println("Running vardict command...")
    command := fmt.Sprintf(`~/biotools/vardict 
    -G %s 
    -f %f 
    -r 4 
    -o 1.5 
    -th %d 
    -L 7000 
    -x 6000 
    -N %s 
    -b %s 
    -c 1 -S 2 -E 3 -g 4 %s | 
    ~/biotools/vardict_app/bin/teststrandbias.R | 
    ~/biotools/vardict_app/bin/var2vcf_valid.pl -A 
    -N %s 
    -E 
    -f %f >%s`,
    *refGenome, *minVaf, *threads, *sampleName, *inputBam, *bedFile, *sampleName, *minVaf, *outputVcf)
	runBashCommand(command)
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