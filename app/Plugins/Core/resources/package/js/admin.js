import axios from "axios";

if(document.getElementById("vue-admin-invitationCode-index")){
    const app = {
        data(){
            return {
                checked: false,
                checkedIds:[],
                show:false,
                arr:[],
                select_arr:[
                    {name:'最新创建',id:1},
                    {name:'最早创建',id:2},
                    {name:'已使用',id:3}
                ],
                selected:selected
            }
        },
        methods: {
            changeAllChecked(){
                if (this.checked) {
                    this.checkedIds = this.arr
                } else {
                    this.checkedIds = []
                }
            },



            // 删除选中
            removeChecked(){
                if(confirm("确定要删除吗? 删除后不可恢复")){
                    axios.post("/admin/Invitation-code/remove",{
                        _token:csrf_token,
                        data:this.checkedIds
                    }).then(r=>{
                        const data = r.data;
                        if(data.success){
                            swal({
                                title:data.result.msg,
                                icon:'success'
                            })
                        }else{
                            swal({
                                title:data.result.msg,
                                icon:'error'
                            })
                        }
                    })
                }
            }
        },
        mounted(){
            axios.post('',{
                _token:csrf_token,
                page:page
            }).then(response =>{
                this.arr=response.data;
            })
        },
        watch:{
            checkedIds(val){
                this.show = val.length > 0;
            },
            checked(val){
                this.show = val === true;
            },
            selected(val){
                location.href="?where="+val
            }
        }
    }
    Vue.createApp(app).mount('#vue-admin-invitationCode-index');
}