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
    String s = "Hello"
    command <<<
        echo ~{s}" "~{name}
    >>>
    output {
       # Write output to standard out
       File output_greeting = stdout()
    }
    runtime {
        container: "ubuntu:latest"
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