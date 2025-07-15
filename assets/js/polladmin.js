let questionCounter = 0;

function addQuestion(questionFromDatabase) {
    questionCounter++;

    let databaseId = 0;
    let questionText = "";
    let questionType = "single-select";
    console.log(questionFromDatabase)
    if (questionFromDatabase) {
        databaseId = questionFromDatabase.ID;
        questionText = questionFromDatabase.text;
        questionType = questionFromDatabase.type;
    }
    const questionsContainer = document.getElementById('questionsContainer');
    const questionDiv = document.createElement('div');
    questionDiv.id = `question-${questionCounter}`;
    questionDiv.dataset.questionId = databaseId;
    questionDiv.classList.add('question-block', 'border', 'border-gray-300', 'p-4', 'rounded-lg', 'bg-gray-50', 'relative');
    questionDiv.innerHTML = `
        <hr/>
        <button type="button" onclick="removeQuestion('${questionDiv.id}')" style="display:inline-block;padding:0;margin:0;background-color:red;width:1.2rem;"                        >&times;</button>
        <h3 style="display:inline-block">Question ${questionCounter}</h3>
        <div class="mb-4">
            <label for="questionText-${questionCounter}">Question Text:</label>
            <input type="text" id="questionText-${questionCounter}" name="questionText" value="${questionText}" required>
        </div>
        <div class="mb-4">
            <label for="questionType-${questionCounter}" >Question Type:</label>
            <select id="questionType-${questionCounter}" name="questionType">
                <option value="single-select" ${questionType == "single-select" ? 'selected': ''}>Single Select</option>
                <option value="multi-select" ${questionType == "multi-select" ? 'selected': ''}>Multi Select</option>
            </select>
        </div>
        <div id="optionsContainer-${questionCounter}" >
            <h4 >Options:</h4>
            <!-- Options will be added here by JavaScript -->
        </div>
        <button type="button" onclick="addOption(${questionCounter})">
            Add Option
        </button>
    `;
    questionsContainer.appendChild(questionDiv);
    if (questionFromDatabase) {
        questionFromDatabase.options.forEach((option) => {
            addOption(questionCounter, option);
        });
    } else {
        addOption(questionCounter,null); // Add at least one option by default
    }
}

function removeQuestion(questionId) {
    document.getElementById(questionId).remove();
}

function addOption(questionNum,optionFromDatabase) {
    console.log(questionNum, optionFromDatabase);
    const  optionIdFromDatabase = optionFromDatabase? optionFromDatabase.ID : 0;
    const optionText = optionFromDatabase? optionFromDatabase.text : '';


    const optionsContainer = document.getElementById(`optionsContainer-${questionNum}`);
    const optionDiv = document.createElement('div');
    const optionHtmlId = `option-${questionNum}-${optionsContainer.children.length}`;
    optionDiv.id = optionHtmlId;
    optionDiv.classList.add('flex', 'items-center', 'space-x-2');
    optionDiv.innerHTML = `
        <button type="button" onclick="removeOption('${optionHtmlId}')" style="display:inline-block;padding:0;margin:0;background-color:red;width:1.2rem;"
                >&times;</button>
        <input type="text" name="optionText" data-option-id="${optionIdFromDatabase || 0}" value="${optionText}" placeholder="Option Text" required style="display:inline-block;width:80%">
    `;
    optionsContainer.appendChild(optionDiv);
}

function removeOption(optionId) {
    document.getElementById(optionId).remove();
}



async function savePoll(pollDatabaseId) {
    pollDatabaseId = pollDatabaseId || 0; // If poll id is provided, use it, otherwise generate a new id.
    const pollTitle = document.getElementById('pollTitle').value;
    const questions = [];

    document.querySelectorAll('.question-block').forEach(qDiv => {
        const questionText = qDiv.querySelector('input[name="questionText"]').value;
        const questionType = qDiv.querySelector('select[name="questionType"]').value;
        const options = [];
        qDiv.querySelectorAll('input[name="optionText"]').forEach(optInput => {
            if (optInput.value.trim() !== '') {
                options.push({
                    DatabaseId: parseInt(optInput.dataset.optionId || 0),  
                    Text: optInput.value
                });
            }
        });

        qid = qDiv.dataset.questionId
        if (!qid){
            qid = 0
        }else{
            qid = parseInt(qid)
        }
        

        if (questionText.trim() !== '' && options.length > 0) {
            questions.push({
                databaseId: qid|| 0,  
                text: questionText,
                type: questionType,
                options: options
            });
        }
    });

    if (questions.length === 0) {
        alert("Please add at least one question with options."); // Using alert for simplicity here, would use custom modal in production
        return;
    }

    try {
        const response = await fetch('/admin/polls/save', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                title: pollTitle,
                questions: questions,
                databaseId: pollDatabaseId // Passing the database ID if editing an existing poll
            })
        });

        const result = await response.json();
        if (response.ok) {
            alert(`Poll created successfully! Poll ID: ${result.pollId}`); // Using alert
            window.location.href = `/admin/polls/edit/${result.pollId}`;
        } else {
            alert(`Error creating poll: ${result.message || response.statusText}`); // Using alert
        }
    } catch (error) {
        console.error('Error:', error);
        alert('An error occurred while creating the poll.'); // Using alert
    }
}

