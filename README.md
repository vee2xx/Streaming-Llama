# Streaming-Llama
A user interface written in Go to submit prompts to OpenAI

## TODO:
1. Server to handle streaming
2. Browser interception/automation?
3. No proprietary interfaces?
4. Easy to deploy IAM


   var script;

    const apiKey = '';
    const response = await fetch('https://api.openai.com/v1/chat/completions', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${apiKey}`
        },
        body: JSON.stringify({
            "model": "gpt-3.5-turbo-16k",
            "messages": [
              {
                "role": "system",
                "content": [
                  {
                    "type": "text",
                    "text": "You are a generation assistant. You will be provided with text describing the problem the script is required to solve. Respond with code only without explanation."
                  }
                ]
              },
              {
                "role": "user",
                "content": [
                  {
                    "type": "text",
                    "text": userPrompt
                  }
                ]
              }
            ],
            "temperature": 1,
            "max_tokens": 100,
            "top_p": 1,
            "frequency_penalty": 0,
            "presence_penalty": 0
          })
    });
    if (response.status == 200) {
        var contents = await response.json();
        script = contents.choices[0].message.content;
    }
    return script
 }