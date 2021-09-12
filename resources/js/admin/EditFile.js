import axios from "axios"
import swal from 'sweetalert';

if(document.getElementById("EditFile")){
    const EditFile = {
        data(){
            return {
                ace:""
            }
        },
        mounted() {
            this.ace = ace.edit("editor");
            this.ace.setTheme("ace/theme/monokai");
            this.ace.session.setMode("ace/mode/"+lang);
            this.ace.session.setUseWrapMode(true);
            this.ace.setHighlightActiveLine(true);
        },
        methods:{
            submit(){
                const content = this.ace.getValue();
                axios.post(action,{
                    _token:csrf_token,
                    content:content
                })
                    .then(r=>{
                        const data = r.data;
                        if(data.success=== true){
                            swal({
                                title:data.result.msg,
                                icon:"success"
                            })
                        }else{
                            swal({
                                title:data.result.msg,
                                icon:"error"
                            })
                        }
                    })
                    .catch(err=>{
                        swal({
                            title:"请求出错,详细查看控制台",
                            icon:"error"
                        })
                        console.error(err);
                    })
            }
        }
    }

    Vue.createApp(EditFile).mount("#EditFile")
}