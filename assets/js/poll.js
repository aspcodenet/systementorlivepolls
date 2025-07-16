

// static/poll.js
document.addEventListener('DOMContentLoaded', () => {
    const pollId = window.location.pathname.split('/').filter(Boolean).pop();
    console.log('Poll ID:', pollId);
    const ws = new WebSocket(`ws://${window.location.host}/ws/${pollId}`);

    const statusMessageDiv = document.getElementById('statusMessage');
    const currentStatusText = document.getElementById('currentStatusText');
    const questionSection = document.getElementById('questionSection');
    const currentQuestionText = document.getElementById('currentQuestionText');
    const questionOptionsDiv = document.getElementById('questionOptions');
    const submitVoteButton = document.getElementById('submitVoteButton');
    const resultsSection = document.getElementById('resultsSection');
    const currentQuestionResultsDiv = document.getElementById('currentQuestionResults');
    const finalResultsSection = document.getElementById('finalResultsSection');
    const allPollResultsDiv = document.getElementById('allPollResults');
    const pollFinishedSection = document.getElementById('pollFinishedSection');

    let currentQuestionData = null; // To store the current question's details

    ws.onopen = (event) => {
        console.log('WebSocket connection opened:', event);
        currentStatusText.textContent = 'Connected to poll. Waiting for poll to start...';
    };

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log('Message from server:', message);



        switch (message.type) {
            case 'poll_state_update':
                updatePollState(message);
                break;
            case 'error':
                alert(`Error: ${message.message}`); // Using alert for simplicity
                break;
            default:
                console.warn('Unknown message type:', message.type);
        }
    };

    ws.onclose = (event) => {
        console.log('WebSocket connection closed:', event);
        currentStatusText.textContent = 'Disconnected from poll. Please refresh.';
        statusMessageDiv.classList.remove('hidden');
        questionSection.classList.add('hidden');
        resultsSection.classList.add('hidden');
        finalResultsSection.classList.add('hidden');
        pollFinishedSection.classList.remove('hidden');
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        currentStatusText.textContent = 'WebSocket error. Please check console.';
        statusMessageDiv.classList.remove('hidden');
    };

    function updatePollState(message) {
        ws.currentPollState = message
        currentStatusText.textContent = `Status: ${message.status.toUpperCase()}`;

        // Hide all dynamic sections initially
        questionSection.classList.add('hidden');
        resultsSection.classList.add('hidden');
        finalResultsSection.classList.add('hidden');
        pollFinishedSection.classList.add('hidden');

        switch (message.status) {
            case 'setup':
                currentStatusText.textContent = 'Waiting for poll to start...';
                break;
            case 'active':
                questionSection.classList.remove('hidden');
                currentQuestionData = message.currentQuestion;
                renderQuestion(currentQuestionData);
                break;
            case 'results':
                questionSection.classList.remove('hidden'); // Still show question text
                resultsSection.classList.remove('hidden');
                currentQuestionData = message.currentQuestion; // Keep current question data
                displayCurrentQuestionResults( currentQuestionData);
                break;
            case 'finished':
                finalResultsSection.classList.remove('hidden');
                pollFinishedSection.classList.remove('hidden');
                displayFinalResults(message.results,message.allQuestions);
                break;
        }
    }

    function renderQuestion(question) {
        console.log(question)
        currentQuestionText.textContent = question.text;
        questionOptionsDiv.innerHTML = ''; // Clear previous options

        question.options.forEach(option => {
            const optionDiv = document.createElement('div');
            optionDiv.classList.add('flex', 'items-center');
            const inputType = question.type === 'single-select' ? 'radio' : 'checkbox';
            const inputName = question.type === 'single-select' ? 'option' : 'options';

            optionDiv.innerHTML = `
                <input type="${inputType}" id="option-${option.ID}" name="${inputName}" value="${option.ID}"
                       class="form-${inputType} h-4 w-4 text-blue-600 border-gray-300 focus:ring-blue-500 rounded">
                <label for="option-${option.ID}" class="ml-2 text-gray-700 text-lg">${option.text}</label>
            `;
            questionOptionsDiv.appendChild(optionDiv);
        });
        submitVoteButton.disabled = false; // Enable submit button
    }

    document.getElementById('voteForm').addEventListener('submit', (event) => {
        event.preventDefault();
        if (!currentQuestionData) {
            alert("No active question to vote on."); // Using alert
            return;
        }

        const selectedOptions = [];
        const inputs = questionOptionsDiv.querySelectorAll('input[name="option"], input[name="options"]');
        inputs.forEach(input => {
            if (input.checked) {
                selectedOptions.push(input.value);
            }
        });

        if (selectedOptions.length === 0) {
            alert("Please select at least one option."); // Using alert
            return;
        }

        if (currentQuestionData.Type === 'single-select' && selectedOptions.length > 1) {
            alert("Please select only one option for this question."); // Using alert
            return;
        }

        if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'submit_vote',
                pollId: pollId,
                questionId: currentQuestionData.ID.toString(),
                selectedOptions: selectedOptions
            }));
            submitVoteButton.disabled = true; // Disable button after submitting vote
            alert("Vote submitted!"); // Using alert
        } else {
            alert('WebSocket not connected. Please refresh the page.'); // Using alert
        }
    });

    function displayCurrentQuestionResults(message) {
        console.log("NU")
        console.log(message)
        
        if (!message.votes) {
            return;
        }


        currentQuestionResultsDiv.innerHTML = '';
        let totalVotes = 0;

        // Find the current question's options from the poll_state_update if available
        // Or, if not, we'll just display the option IDs.
        let currentQuestionOptionsMap = new Map();
        if (ws.currentPollState && ws.currentPollState.currentQuestion) {

            ws.currentPollState.currentQuestion.options.forEach(opt => {
                currentQuestionOptionsMap.set(opt.ID, opt.text);
            });
        }

   

        // Sort votes by count descending for better visualization
        const sortedOptions = Object.keys(message.votes).sort((a, b) => message.votes[b] - message.votes[a]);
        console.log(sortedOptions)        



        sortedOptions.forEach(optionId => {
            const count = message.votes[optionId];
            totalVotes += count;

            const optionText = currentQuestionOptionsMap.has(parseInt(optionId)) ? currentQuestionOptionsMap.get(parseInt(optionId)) : `Option ${optionId.substring(0, 8)}`;
            console.log(optionText)

            const voteItem = document.createElement('div');
            voteItem.classList.add('mb-2');
            voteItem.innerHTML = `
                <div class="flex justify-between items-center mb-1">
                    <span class="text-gray-800 font-medium">${optionText}</span>
                    <span class="text-gray-600">${count} votes</span>
                </div>
                <div class="vote-bar-container">
                    <div class="vote-bar" style="width: ${totalVotes > 0 ? (count / totalVotes * 100) : 0}%"></div>
                </div>
            `;
            currentQuestionResultsDiv.appendChild(voteItem);
        });
        //totalVotesSpan.textContent = totalVotes;
    }

    function displayFinalResults(allResults, allQuestions) {
        allPollResultsDiv.innerHTML = '';

        // This client-side code assumes it has access to the full poll structure
        // to map question IDs to text and option IDs to text.
        // For this example, we'll use a simplified approach for demonstration.
        // In a real app, the poll structure would be loaded initially or sent with updates.

        for (const questionId in allResults) {
            const questionResults = allResults[questionId];
            const questionBlock = document.createElement('div');
            questionBlock.classList.add('border', 'border-purple-300', 'p-4', 'rounded-lg', 'bg-purple-50', 'mb-4');

            currentQ = allQuestions.find(el=>el.ID == questionId)
            let questionText = currentQ.text
            let questionOptionsMap = new Map();

            currentQ.options.forEach(opt => questionOptionsMap.set(opt.ID, opt.text));

            questionBlock.innerHTML = `<h3>${questionText}</h3>`;
            const resultsList = document.createElement('div');
            resultsList.classList.add('space-y-2');

            let totalVotesForQuestion = 0;
            for (const optionId in questionResults) {
                totalVotesForQuestion += questionResults[optionId];
            }

            console.log(questionOptionsMap)

            const sortedOptions = Object.keys(questionResults).sort((a, b) => questionResults[b] - questionResults[a]);

            sortedOptions.forEach(optionId => {
                const count = questionResults[optionId];
                const percentage = totalVotesForQuestion > 0 ? ((count / totalVotesForQuestion) * 100).toFixed(1) : 0;
                const optionText = questionOptionsMap.get(parseInt(optionId));
                

                const resultItem = document.createElement('div');
                resultItem.classList.add('mb-2');
                resultItem.innerHTML = `
                    <div class="votes-flex-container">
                        <span>${optionText}</span>
                        <span>${count} votes (${percentage}%)</span>
                    </div>
                    <div class="vote-bar-container">
                        <div class="vote-bar" style="width: ${percentage}%"></div>
                    </div>
                `;
                resultsList.appendChild(resultItem);
            });
            questionBlock.appendChild(resultsList);
            allPollResultsDiv.appendChild(questionBlock);
        }
    }
});
