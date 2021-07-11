if(document.getElementById("vue-user-create-class")){
    const vucc = {
        data(){
            return {
                name:"",
                icon:"",
                color:"#206bc4",
                quanxian:1
            }
        },
        methods:{
            submit(){

            }
        }
    }
    Vue.createApp(vucc).mount("#vue-user-create-class")
}