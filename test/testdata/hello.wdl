version 1.1

workflow HelloWorld {
    input {
        String wf_input_name
    }
    call WriteGreeting { input: name = wf_input_name }
    output {
        File wf_output_greeting = WriteGreeting.output_greeting
    }
    meta {
        author: "Yunhai Luo"
        version: 1.1
        for: "workflow"
    }
    parameter_meta {
        name: {
            help: "A name for workflow input"
        }
    }
}

task WriteGreeting {
    input {
        String name
    }
    command <<<
       echo "Hello "~{name}
    >>>
    output {
       # Write output to standard out
       File output_greeting = stdout()
    }
    meta {
        author: "Yunhai Luo"
        version: 1.1
        for: "task"
    }
    parameter_meta {
        name: {
            help: "One name as task input"
        }
    }
}