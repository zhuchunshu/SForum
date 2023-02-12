<div class="row row-cards" id="vue-user-my-setting">
    <div class="col-md-4">
        <div class="row row-cards">
            <div class="col-md-12">
                <form action="/user/myUpdate?Redirect=/user/setting" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">基本设置</div>
                        <div class="mb-3">
                            <label class="form-label">用户名</label>
                            <input disabled type="text" v-model="username" class="form-control">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">邮箱</label>
                            <input disabled type="text" value="{{auth()->data()->email}}" class="form-control">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">用户组</label>
                            {!! Core_Ui()->Html()->UserGroup($data->Class) !!}
                        </div>
                        <div class="mb-3">
                            <label class="form-label">旧密码</label>
                            <input type="text" v-model="old_pwd" name="old_pwd" class="form-control">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">新密码</label>
                            <input type="text" v-model="new_pwd" name="new_pwd" class="form-control">
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary" type="submit">提交</button>
                        </div>
                    </div>
                </form>
            </div>
            <div class="col-md-12">
                <form action="/user/myUpdate/options?Redirect=/user/setting" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">其他设置</div>
                        <div class="mb-3">
                            <label class="form-label">{{__("app.bio")}}</label>
                            <textarea class="form-control" name="qianming" type="text">{{$data->options->qianming}}</textarea>
                        </div>
                        <div class="mb-3">
                            <label class="form-label">QQ</label>
                            <input class="form-control" type="text" placeholder="" name="qq" value="{{$data->options->qq}}">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">微信</label>
                            <input class="form-control" type="text" placeholder="" name="wx" value="{{$data->options->wx}}">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">个人网站</label>
                            <input class="form-control" type="url" placeholder="" name="website" value="{{$data->options->website}}">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">对外展示邮箱</label>
                            <input class="form-control" type="email" placeholder="" name="email" value="{{$data->options->email}}">
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary" type="submit">提交</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
    <div class="col-md-4">
        <div class="row row-cards">
            <div class="col-md-12">
                <form action="/user/myUpdate/avatar?Redirect=/user/setting" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">修改头像</div>
                        <div class="mb-3">
                            <label class="form-label">当前头像
                                <div><span class="avatar" style="background-image: url({{super_avatar(auth()->data())}})"></span></div>
                            </label>
                        </div>
                        <div class="mb-3">
                            <label class="form-label">选择头像</label>
                            <input name="avatar" type="file" accept="image/gif, image/png, image/jpeg, image/jpg" class="form-control">
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary" type="submit">提交</button>
                        </div>
                    </div>
                </form>
            </div>

            <div class="col-md-12">
                <form action="/user/myUpdate/noticed?Redirect=/user/setting" method="post">
                    <x-csrf/>
                    <div class="card card-body">
                        <div class="card-title">通知设置</div>
                        <div class="mb-3">
                            <label class="form-check form-switch">
                                <input name="email" class="form-check-input" type="checkbox" @if((string)get_user_settings(auth()->id(), 'noticed')==="1") {{"checked"}} @endif>
                                <span class="form-check-label">@if(get_options('user_email_noticed_on','false')==="true")接收邮件通知@else不接收邮件通知@endif</span>
                            </label>
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary" type="submit">提交</button>
                        </div>
                    </div>
                </form>
            </div>

        </div>
    </div>
    <div class="col-md-4">
        <form action="/user/myUpdate/other?Redirect=/user/setting" method="post">
            <x-csrf/>
            <div class="card card-body">
                <div class="card-title">其他</div>
                <div class="mb-3">
                    <button class="btn btn-primary" type="submit" name="action" value="removeAvatar">移除头像</button>
                </div>
            </div>
        </form>
    </div>
</div>