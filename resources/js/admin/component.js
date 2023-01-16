import axios from "axios";
import swal from "sweetalert";

if (document.getElementById('vue-admin-hook-component-edit')) {
    const app = {
        data() {
            return {
                ace: ""
            }
        },
        mounted() {
            this.ace = ace.edit("editor");
            this.ace.setTheme("ace/theme/one_dark");
            this.ace.session.setMode("ace/mode/" + lang);
            this.ace.session.setUseWrapMode(true);
            this.ace.setHighlightActiveLine(true);
            this.ace.setShowPrintMargin(false)
            this.ace.setOptions({
                wrap: true,
                enableBasicAutocompletion: true,
                enableSnippets: true,
                enableLiveAutocompletion: true,
                autoScrollEditorIntoView: true
            });
            axios.get(get_action).then(r => {
                const data = r.data;
                if (data.success === true) {
                    this.ace.session.setValue(data.result.content)
                    return ;
                }
                swal({
                    title: data.result.msg,
                    icon: "error"
                })
            })
                .catch(err => {
                    swal({
                        title: "请求出错,详细查看控制台",
                        icon: "error"
                    })
                    console.error(err);
                })
        },
        methods: {
            submit() {
                const content = this.ace.getValue();
                axios.post(put_action,{
                    _token:csrf_token,
                    content:content
                })
                    .then(r => {
                        const data = r.data;
                        if (data.success === true) {
                            swal({
                                title: data.result.msg,
                                icon: "success"
                            })
                        } else {
                            swal({
                                title: data.result.msg,
                                icon: "error"
                            })
                        }
                    })
                    .catch(err => {
                        swal({
                            title: "请求出错,详细查看控制台",
                            icon: "error"
                        })
                        console.error(err);
                    })
            }
        }
    }

    Vue.createApp(app).mount("#vue-admin-hook-component-edit")
}


// 创建小部件
if (document.getElementById('vue-admin-hook-component-create')) {
    const app = {
        data() {
            return {
                name:null,
                ace: ""
            }
        },
        mounted() {
            this.ace = ace.edit("editor");
            this.ace.setTheme("ace/theme/one_dark");
            this.ace.session.setMode("ace/mode/" + lang);
            this.ace.session.setUseWrapMode(true);
            this.ace.setHighlightActiveLine(true);
            this.ace.setShowPrintMargin(false)
            this.ace.setOptions({
                wrap: true,
                enableBasicAutocompletion: true,
                enableSnippets: true,
                enableLiveAutocompletion: true,
                autoScrollEditorIntoView: true
            });
            this.ace.setValue("{{--第一行注释里的内容会被加载为备注--}}\n")
        },
        methods: {
            submit() {
                if(!this.name){
                    swal('Error','小部件名称不能为空','error')
                    return ;
                }
                const content = this.ace.getValue();
                axios.post(create_action,{
                    _token:csrf_token,
                    name:this.name,
                    content:content
                })
                    .then(r => {
                        const data = r.data;
                        if (data.success === true) {
                            swal({
                                title: data.result.msg,
                                icon: "success"
                            })
                            setTimeout(()=>{
                                location.href=data.result.redirect
                            },1500)
                        } else {
                            swal({
                                title: data.result.msg,
                                icon: "error"
                            })
                        }
                    })
                    .catch(err => {
                        swal({
                            title: "请求出错,详细查看控制台",
                            icon: "error"
                        })
                        console.error(err);
                    })
            }
        }
    }

    Vue.createApp(app).mount("#vue-admin-hook-component-create")
}

if(document.getElementById('vue-admin-hook-components')){
    const app ={
        methods:{
            data(){
                return {

                }
            },
            rm(path){
                swal({
                    title: "Are you sure?",
                    text: "一经删除不可恢复,确定要删除吗？",
                    icon: "warning",
                    buttons: true,
                    dangerMode: true,
                })
                    .then((willDelete) => {
                        if (willDelete) {
                            axios.post("/admin/hook/components/delete",{
                                _token:csrf_token,
                                path:path
                            }).then(r=>{
                                const data = r.data
                                if(data.success===true){
                                    swal('Success',data.result.msg,'success')
                                    setTimeout(()=>{
                                        location.reload()
                                    },1000)
                                }else{
                                    swal('Error',data.result.msg,'error')
                                }
                            }).catch(e=>{
                                swal("请求出错", {
                                    icon: "error",
                                });
                                console.error(e)
                            })
                        }
                    });
                console.log(path)
            }
        }
    }
    Vue.createApp(app).mount("#vue-admin-hook-components")
}