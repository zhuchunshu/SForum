@extends("plugins.Core.app")
@section('title','个人设置')
@section('content')
    <div id="vue-user-my-setting">
            <div class="row row-cards">
                <div class="col-md-4">
                    <form action="/user/myUpdate?Redirect=/user/setting" method="post" enctype="multipart/form-data">
                        <x-csrf/>
                        <div class="card card-body">
                            <div class="card-title">基本设置</div>
                            <div class="mb-3">
                                <label class="form-label">用户名</label>
                                <input disabled type="text" v-model="username" class="form-control">
                            </div>
                            <div class="mb-3">
                                <label class="form-label">用户组</label>
                                {!! Core_Ui()->Html()->UserGroup($data->Class) !!}
                            </div>
                            <div class="mb-3">
                                <label class="form-label">邮箱</label>
                                <input disabled type="email" v-model="email" class="form-control">
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
                <div class="col-md-4">
                    <form action="/user/myUpdate/avatar?Redirect=/user/setting" method="post" enctype="multipart/form-data">
                        <x-csrf/>
                        <div class="card card-body">
                            <div class="card-title">修改头像</div>
                            <div class="mb-3">
                                <label class="form-label">当前头像
                                    <div>{!! avatar(auth()->data()->id) !!}</div>
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
            </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection