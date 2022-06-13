
// 私信
class OwO {
    constructor(option) {
        const defaultOption = {
            logo: 'OwO表情',
            container: document.getElementsByClassName('OwO')[0],
            target: document.getElementsByTagName('textarea')[0],
            position: 'down',
            width: '100%',
            maxHeight: '250px',
            api: 'https://api.anotherhome.net/OwO/OwO.json'
        };
        for (let defaultKey in defaultOption) {
            if (defaultOption.hasOwnProperty(defaultKey) && !option.hasOwnProperty(defaultKey)) {
                option[defaultKey] = defaultOption[defaultKey];
            }
        }
        this.container = option.container;
        this.target = option.target;
        if (option.position === 'up') {
            this.container.classList.add('OwO-up');
        }

        const xhr = new XMLHttpRequest();
        xhr.onreadystatechange = () => {
            if (xhr.readyState === 4) {
                if (xhr.status >= 200 && xhr.status < 300 || xhr.status === 304) {
                    this.odata = JSON.parse(xhr.responseText);
                    this.init(option);
                }
                else {
                    console.log('OwO data request was unsuccessful: ' + xhr.status);
                }
            }
        };
        xhr.open('get', option.api, true);
        xhr.send(null);
    }

    init(option) {
        this.area = option.target;
        this.packages = Object.keys(this.odata);

        // fill in HTML
        let html = `
            <div class="OwO-logo"><span>${option.logo}</span></div>
            <div class="OwO-body" style="width: ${option.width}">`;

        for (let i = 0; i < this.packages.length; i++) {

            html += `
                <ul class="OwO-items OwO-items-${this.odata[this.packages[i]].type}" style="max-height: ${parseInt(option.maxHeight) - 53 + 'px'};">`;

            let opackage = this.odata[this.packages[i]].container;
            for (let i = 0; i < opackage.length; i++) {

                html += `
                    <li class="OwO-item" title="${opackage[i].text}">${opackage[i].icon}</li>`;

            }

            html += `
                </ul>`;
        }

        html += `
                <div class="OwO-bar">
                    <ul class="OwO-packages">`;

        for (let i = 0; i < this.packages.length; i++) {

            html += `
                        <li><span>${this.packages[i]}</span></li>`

        }

        html += `
                    </ul>
                </div>
            </div>
            `;
        this.container.innerHTML = html;

        // bind event
        this.logo = this.container.getElementsByClassName('OwO-logo')[0];
        this.logo.addEventListener('click', () => {
            this.toggle();
        });

        this.container.getElementsByClassName('OwO-body')[0].addEventListener('click', (e)=> {
            let target = null;
            if (e.target.classList.contains('OwO-item')) {
                target = e.target;
            }
            else if (e.target.parentNode.classList.contains('OwO-item')) {
                target = e.target.parentNode;
            }
            if (target) {
                const cursorPos = this.area.selectionEnd;
                let areaValue = this.area.value;

                const tag = e.target.getElementsByTagName('img');
                if(e.target.nodeName!=="IMG"){
                    if(tag.length>0){
                        this.area.value = areaValue.slice(0, cursorPos) + " " + e.target.title + " " + areaValue.slice(cursorPos);
                    }else{
                        this.area.value = areaValue.slice(0, cursorPos) + " " + target.innerHTML + " " + areaValue.slice(cursorPos);
                    }
                }else{
                    this.area.value = areaValue.slice(0, cursorPos) + " " + e.target.parentElement.title +" " + areaValue.slice(cursorPos);
                }
                this.area.focus();
                this.toggle();
            }
        });

        this.packagesEle = this.container.getElementsByClassName('OwO-packages')[0];
        for (let i = 0; i < this.packagesEle.children.length; i++) {
            ((index) =>{
                this.packagesEle.children[i].addEventListener('click', () => {
                    this.tab(index);
                });
            })(i);
        }

        this.tab(0);
    }

    toggle() {
        if (this.container.classList.contains('OwO-open')) {
            this.container.classList.remove('OwO-open');
        }
        else {
            this.container.classList.add('OwO-open');
        }
    }

    tab(index) {
        const itemsShow = this.container.getElementsByClassName('OwO-items-show')[0];
        if (itemsShow) {
            itemsShow.classList.remove('OwO-items-show');
        }
        this.container.getElementsByClassName('OwO-items')[index].classList.add('OwO-items-show');

        const packageActive = this.container.getElementsByClassName('OwO-package-active')[0];
        if (packageActive) {
            packageActive.classList.remove('OwO-package-active');
        }
        this.packagesEle.getElementsByTagName('li')[index].classList.add('OwO-package-active');
    }
}
if (typeof module !== 'undefined' && typeof module.exports !== 'undefined') {
    module.exports = OwO;
}
else {
    window.OwO = OwO;
}



// pm
if(document.getElementById('user-pm-container')){
    const app = {
        data(){
            return {
                socket:null,
                msg:null,
                to_id:to_id,
                btn:{
                    disabled:false
                },
                messages:0
            }
        },
        mounted(){
            this.InitOwO();
            this.InitSocket();
            setInterval(()=>{
                this.msg = document.getElementsByTagName('textarea')[0].value;
            },300);
        },
        methods:{

            InitOwO(){
                new OwO({
                    logo: '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-mood-smile" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">\n' +
                        '                                                <desc>Download more icon variants from https://tabler-icons.io/i/mood-smile</desc>\n' +
                        '                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>\n' +
                        '                                                <circle cx="12" cy="12" r="9"></circle>\n' +
                        '                                                <line x1="9" y1="10" x2="9.01" y2="10"></line>\n' +
                        '                                                <line x1="15" y1="10" x2="15.01" y2="10"></line>\n' +
                        '                                                <path d="M9.5 15a3.5 3.5 0 0 0 5 0"></path>\n' +
                        '                                            </svg>',
                    container: document.getElementsByClassName('OwO')[0],
                    target: document.getElementsByClassName('OwO-textarea')[0],
                    api: '/api/core/OwO.json',
                    position: 'down',
                    width: '100%',
                    maxHeight: '250px'
                })
            },
            // 初始化socket
            InitSocket(){
                this.socket = io(pm_socket, { transports: ["websocket"] });
                this.socket.on("connect", () => {
                    if(this.socket.connected===false){
                        swal({
                            title:"聊天室连接失败!",
                            icon: "error"
                        })
                        return ;
                    }
                    this.socket.emit('join-room', '{"token":"'+ _token+'","to_id":"'+to_id+'"}');
                    setInterval(()=>{
                        this.socket.emit('getMsg','{"token":"'+ _token+'","to_id":"'+to_id+'"}');
                        this.socket.on('getMsg',(data)=>{
                            this.messages = data;
                        });
                    }, 3000);

                });

            },
            // 发消息
            sendMsg(){
                console.log(this.msg)
                if(!this.msg){
                    swal({
                        title:"不能发送空消息",
                        icon : "error"
                    })
                    return ;
                }
                this.btn.disabled = this.btn.disabled === false;
                if(this.socket.emit('sendMsg','{"token":"'+ _token+'","to_id":"'+to_id+'","msg" : "'+this.msg+'"}')){
                    this.btn.disabled = this.btn.disabled === false;
                    this.msg = null;
                    location.reload();
                }
            }
        }
    }

    Vue.createApp(app).mount('#user-pm-container');
}

$('#chat-list').scrollTop($("#chat-list")[0].scrollHeight)