<div class="row row-cards" id="vue-user-my-setting">
    <div class="col-md-12">
        <form action="/user/setbackgroundImg?Redirect=/user/setting?m=userSetting_3" method="post" enctype="multipart/form-data">
            <x-csrf/>
            <div class="card card-body">
                <div class="card-title">主页背景图</div>
                <img src="{{get_user_settings(auth()->id(),'backgroundImg','https://images.unsplash.com/photo-1653185053677-26b081c1a214?ixlib=rb-1.2.1&ixid=MnwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8&auto=format&fit=crop&w=1632&q=80')}}" alt="">
                <div class="mt-3">
                    <input type="file" accept="image/png, image/jpeg, image/jpg" class="form-control" name="backgroundImg" required>
                </div>
                <div class="mb-3 mt-3">
                    <button class="btn btn-primary" type="submit">提交</button>
                </div>
            </div>
        </form>
    </div>
</div>