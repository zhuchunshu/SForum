<div class="row row-cards" id="vue-user-my-setting">
    <div class="col-md-12">
        <form action="/user/setbackgroundImg?Redirect=/user/setting?m=userSetting_3" method="post" enctype="multipart/form-data">
            <x-csrf/>
            <div class="card card-body">
                <div class="card-title">主页背景图</div>
                <img src="{{get_user_settings(auth()->id(),'backgroundImg','/plugins/Core/image/user_background.jpg')}}" alt="">
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