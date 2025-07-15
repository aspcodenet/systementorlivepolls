document.addEventListener('DOMContentLoaded', () => {
    // get last part of the URL to get the poll ID from the query parameters, e.g., /poll/1234567890
    const pollId = window.location.pathname.split('/').filter(Boolean).pop();
    console.log('Poll ID:', pollId);

    const ws = new WebSocket(`ws://${window.location.host}/ws/${pollId}?role=admin`);

    const currentStatusText = document.getElementById('currentStatusText');
    const questionSection = document.getElementById('questionSection');
    const currentQuestionText = document.getElementById('currentQuestionText');
    const currentQuestionOptions = document.getElementById('currentQuestionOptions');
    const realtimeResultsDiv = document.getElementById('realtimeResults');
    const voteCountsDiv = document.getElementById('voteCounts');
    const totalVotesSpan = document.getElementById('totalVotes');
    const finalResultsDiv = document.getElementById('finalResults');
    const allPollResultsDiv = document.getElementById('allPollResults');

    const startButton = document.getElementById('startButton');
    const nextButton = document.getElementById('nextButton');
    const showResultsButton = document.getElementById('showResultsButton');
    const doneButton = document.getElementById('doneButton');

    ws.onopen = (event) => {
        console.log('WebSocket connection opened:', event);
        currentStatusText.textContent = 'Connected to poll. Waiting for updates...';
    };

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        console.log('Message from server:', message);

        switch (message.type) {
            case 'poll_state_update':
                ws.currentPollState = message
                updatePollState(message);
                break;
            case 'admin_results_update':
                updateRealtimeResults(message);
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
        startButton.classList.add('hidden');
        nextButton.classList.add('hidden');
        showResultsButton.classList.add('hidden');
        doneButton.classList.add('hidden');
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        currentStatusText.textContent = 'WebSocket error. Please check console.';
    };

    function updatePollState(message) {
        currentStatusText.textContent = `Status: ${message.status.toUpperCase()}`;

        // Hide all dynamic sections initially
        questionSection.classList.add('hidden');
        realtimeResultsDiv.classList.add('hidden');
        finalResultsDiv.classList.add('hidden');

        // Hide all buttons initially
        startButton.classList.add('hidden');
        nextButton.classList.add('hidden');
        showResultsButton.classList.add('hidden');
        doneButton.classList.add('hidden');

        switch (message.status) {
            case 'setup':
                startButton.classList.remove('hidden');
                currentStatusText.textContent = 'Poll is in setup mode. Click Start Poll to begin.';
                break;
            case 'active':
                questionSection.classList.remove('hidden');
                realtimeResultsDiv.classList.remove('hidden'); // Admin always sees real-time results
                nextButton.classList.remove('hidden');
                showResultsButton.classList.remove('hidden');

                if (message.currentQuestion) {
                    currentQuestionText.textContent = message.currentQuestion.text;
                    currentQuestionOptions.innerHTML = '';
                    message.currentQuestion.options.forEach(option => {
                        const p = document.createElement('p');
                        p.classList.add('text-gray-700', 'text-lg', 'ml-4');
                        p.textContent = `- ${option.text}`;
                        currentQuestionOptions.appendChild(p);
                    });
                }
                break;
            case 'results':
                questionSection.classList.remove('hidden');
                realtimeResultsDiv.classList.remove('hidden'); // Admin always sees real-time results
                nextButton.classList.remove('hidden');
                doneButton.classList.remove('hidden'); // Admin can mark poll done from results
                
                if (message.currentQuestion) {
                    currentQuestionText.textContent = message.currentQuestion.text;
                    currentQuestionOptions.innerHTML = ''; // Clear options, as results are shown in realtimeResultsDiv
                }
                // The realtimeResultsDiv will be updated by admin_results_update
                break;
            case 'finished':
                finalResultsDiv.classList.remove('hidden');
                displayFinalResults(message.results, message.allQuestions); // Pass current question for context
                currentStatusText.textContent = 'Poll has finished. Final results are displayed.';
                break;
        }
    }

    function updateRealtimeResults(message) {
        console.log("NU")
        console.log(message)
        
        if (!message.votes || !message.questionId) {
            console.warn("Invalid admin_results_update message:", message);
            return;
        }

        voteCountsDiv.innerHTML = '';
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
        console.log(currentQuestionOptionsMap)        

        sortedOptions.forEach(optionId => {
            const count = message.votes[optionId];
            totalVotes += count;

            const optionText = currentQuestionOptionsMap.has(parseInt(optionId)) ? currentQuestionOptionsMap.get(parseInt(optionId)) : `Option ${optionId.substring(0, 8)}`;

            const voteItem = document.createElement('div');
            voteItem.innerHTML = `
                <div class="votes-flex-container">
                    <span>${optionText}</span>
                    <span>${count} votes</span>
                </div>
                <div class="vote-bar-container">
                    <div class="vote-bar" style="width: ${totalVotes > 0 ? (count / totalVotes * 100) : 0}%"></div>
                </div>
            `;
            voteCountsDiv.appendChild(voteItem);
        });
        totalVotesSpan.textContent = totalVotes;
    }

    function displayFinalResults(allResults, allQuestions) {
        allPollResultsDiv.innerHTML = '';

        // Iterate through each question's results
        for (const questionId in allResults) {
            const questionResults = allResults[questionId];
            const questionBlock = document.createElement('div');
            questionBlock.classList.add('border', 'border-purple-300', 'p-4', 'rounded-lg', 'bg-purple-50', 'mb-4');

            currentQ = allQuestions.find(el=>el.ID == questionId)
            let questionText = currentQ.text
            let questionOptionsMap = new Map();

            currentQ.options.forEach(opt => questionOptionsMap.set(opt.ID, opt.text));


            questionBlock.innerHTML = `<h3 class="text-xl font-semibold mb-3 text-purple-800">${questionText}</h3>`;
            const resultsList = document.createElement('div');
            resultsList.classList.add('space-y-2');

            let totalVotesForQuestion = 0;
            for (const optionId in questionResults) {
                totalVotesForQuestion += questionResults[optionId];
            }

            // Sort options by vote count for the final display
            const sortedOptions = Object.keys(questionResults).sort((a, b) => questionResults[b] - questionResults[a]);

            sortedOptions.forEach(optionId => {
                const count = questionResults[optionId];
                const percentage = totalVotesForQuestion > 0 ? ((count / totalVotesForQuestion) * 100).toFixed(1) : 0;
                const optionText = questionOptionsMap.has(optionId) ? questionOptionsMap.get(optionId) : `Option ${optionId.substring(0, 8)}`;

                const resultItem = document.createElement('div');
                resultItem.classList.add('mb-2');
                resultItem.innerHTML = `
                    <div class="flex justify-between items-center mb-1">
                        <span class="text-gray-800 font-medium">${optionText}</span>
                        <span class="text-gray-600">${count} votes (${percentage}%)</span>
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

    window.sendAdminAction = (action) => {
        if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'admin_action',
                pollId: pollId,
                action: action
            }));
        } else {
            alert('WebSocket not connected. Please refresh the page.'); // Using alert
        }
    };
});
