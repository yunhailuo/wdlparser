version 1.1

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