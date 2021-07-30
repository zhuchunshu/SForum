const header_left ={
    methods: {
        reload(){
            location.reload();
        }
    },
}

Vue.createApp(header_left).mount("#vue-header-left")