<div class="border border-danger text-center">
    <div class="p-2">
        <h5 class="text-danger">购买提示</h5>
        <p class="text-muted">购买后可查看隐藏内容</p>
    </div>
    @yield('buy_content')
    <div class="p-2">
        <button onclick="pay()" class="btn btn-danger">立即购买</button>
    </div>
</div>

<script>
    function pay() {
        // swal 询问是否购买
        swal({
            title: "确认购买?",
            icon: "warning",
            buttons: true,
            dangerMode: true,
        }).then((willBuy) => {
            if (willBuy) {
                // 购买
                // fetch 发送post请求
                fetch("/api/topic/shortcode/buy.post", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                        "Accept": "application/json",
                    },
                    body: JSON.stringify({
                        "post_id": "{{ $post_id }}",
                        _token: "{{ csrf_token() }}"
                    })
                }).then(response => response.json())
                    .then(data => {
                        if(data.success===true){
                            swal("购买成功", {
                                icon: "success",
                            }).then(()=>{
                                window.location.reload()
                            })
                            return ;
                        }
                        if(data.result.url){
                            swal({
                                title:"购买失败",
                                text: data.result.msg,
                                icon: "error",
                            }).then(()=>{
                                window.open(data.result.url)
                            })
                        }else{
                            swal({
                                title:"购买失败",
                                text: data.result.msg,
                                icon: "error",
                            })
                        }
                    })
            }
        });
    }
</script>