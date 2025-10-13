YAML-to-README
Original Problem to Solve
The original problem I was trying to address when it came to our telco-reference repo was that I didn't know exactly what I was applying to my OpenShift cluster by just reading the YAML file names as they do not necessarily contain the entire substance of the MachineConfig object and what it does in the name.  

There are hundreds of YAMLs in this repo that live in multiple different folders and to go through them all individually would be a large undertaking.  The top-level README files were either lacking information or essentially placeholders.  There was a good opportunity here to add additional context for users of the telco-reference repo by creating nice READMEs for everyone to reference.

The Solution
What I was able to do was to create a Golang-based application that traverses a folder and underlying folders looking for YAML files and then passing their contents into a model running locally on my Mac.  In this case, I settled on using ollama and llama3.2:latest as my model of choice.  I tried to use other models but was not able to get the same quality of strings generated as I was with llama3.2:latest.

Ollama running locally on my Mac made it very easy to interact with the model and their Golang library is well-documented so I was able to quickly leverage it for this project.

For my application, I decided to make it a Golang-based CLI application using the cobra CLI library.  One of the more difficult parts of developing this application was actually the prompt engineering for the prompt passed along with the YAML contents.  I originally told it to create two or three sentences to summarize the YAMLs contents but it needed way more guidance than that.  The final prompt ended up looking like:

SummarizePrompt = "Summarize the purpose of this YAML file in no more than two short, high-level sentences. Do not include any lists, breakdowns, explanations, advice, notes, or formatting. Do not use markdown. No newlines. No code sections. Only output a single, concise summary of the file's purpose, and nothing else. Stop after two sentences. If you cannot summarize in two sentences, summarize in one: \n"

I also built some logic in my yaml-to-readme application where if a YAML file had already been summarized in a prior run, then it would no-op and not regenerate a new string to prevent potential automated runs from regenerating perfectly good summaries over and over again.

My pull request is still open for review for adding these newly created summary READMEs to the repository.

Notable Knowledge Gained:
I tried running the application on Github Actions free tier runners with a local instance of ollama running there and the runtime went from 4 minutes on my local M3 Max Macbook Pro to 4 hours.  You can look at the output of my test runs here.  Technically it was possible to generate all of the YAML summaries with Github Actions but it was painfully slow.

I also learned a lot about Ollama in general.  At home I have a RTX 3080 GPU on one of my other PCs and I setup Ollama there and allowed for LAN traffic to be able to hit that GPU to handle my workload.  I learned that my M3 Max CPU was just as fast or even slightly faster than my RTX 3080 at generating the summaries.

​Reference Links:

https://github.com/openshift-kni/telco-reference/pull/220
https://github.com/sebrandon1/yaml-to-readme