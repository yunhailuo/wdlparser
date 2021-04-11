version 1.1

import "9errors.wdl"
import "http://example.com/lib/analysis_tasks" as analysis
import "https://example.com/lib/stdlib.wdl"

workflow HelloWorld {
    call WriteGreeting
}

task WriteGreeting {
    command {
       echo "Hello World"
    }
    output {
       # Write output to standard out
       File output_greeting = stdout()
    }
}