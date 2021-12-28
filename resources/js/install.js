import axios from "axios";

if(document.getElementById("app-install")){
    $(function(){

        // 初始化
        function init(data=null){
            if(data===null){
                axios.post("/install",{
                    _token:csrf_token
                }).then(r=>{
                    $("#app-install").html(r.data)
                })
            }else{
                axios.post("/install",data).then(r=>{
                    $("#app-install").html(r.data)
                })
            }
        }
        // 先初始化
        init();

        // 下一步
        function next() {
            var d = {};
            var t = $('form').serializeArray();
            $.each(t, function() {
                d[this.name] = this.value;
            });
            init(d)
            return false;
        }

        // 上一步
        function prev(){
            init({
                _token:csrf_token,
                reduce:true,
            })
        }

        $("form").submit(function(){
            next()
        })

        $("#prev").click(function(){
            prev()
        })

    })


}
