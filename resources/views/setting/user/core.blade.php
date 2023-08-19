<div class="card card-body">
    <div class="row">
        <div class="col-4">
            <label class="form-label">默认头像源
            </label>
            <select class="form-select" v-model="data.core_user_def_avatar">
                <option value="gavatar">Gavatar</option>
                <option value="ui-avatars">ui-avatars</option>
            </select>
            <small>默认使用Gravatar的头像</small>
        </div>

        <div class="col-1">
            <label class="form-label">ui-avatars
                <div>
                    <span class="avatar" style="background-image:url(https://ui-avatars.com/api/?background=random&name=Zhu+Chunshu)"></span>
                </div>
            </label>
        </div>

        <div class="col-4">
            <label class="form-label">头像缓存
            </label>
            <select class="form-select" v-model="data.core_user_avatar_cache">
                <option value="1">开启</option>
                <option value="2">关闭</option>
            </select>
            <small>默认开启</small>
        </div>

        <div class="col-3">
            <label class="form-label">强制验证邮箱
            </label>
            <select class="form-select" v-model="data.core_user_email_ver">
                <option value="1">开启</option>
                <option value="2">关闭</option>
            </select>
            <small>默认开启</small>
        </div>


        <div class="col-3">
            <label class="form-label">头像缓存时间
            </label>
            <input type="number" class="form-control" v-model="data.core_user_def_avatar_cache">
            <small>头像缓存时间/秒,减少查询,大幅度优化速度 默认10分钟</small>
        </div>

        <div class="col-3">
            <label class="form-label">图片上传大小限制 / KB
            </label>
            <input type="number" class="form-control" v-model="data.core_user_up_img_size">
            <small>当前:{{get_options('core_user_up_img_size',2048)}} KB</small>
        </div>

        <div class="col-3">
            <label class="form-label">文件上传大小限制 / KB
            </label>
            <input type="number" class="form-control" v-model="data.core_user_up_file_size">
            <small>当前:{{get_options('core_user_up_file_size',4096)}} KB</small>
        </div>
        <div class="col-3">
            <label class="form-label">允许几台设备同时在线？
            </label>
            <input type="number" class="form-control" v-model="data.core_user_session_num">
            <small>当前:{{get_options('core_user_session_num',10)}}</small>
        </div>

        <div class="col-3 align-self-center">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.user_email_noticed_on">
                <span class="form-check-label">关闭所有用户邮件通知功能(需用户自行手动开启)</span>
            </label>
        </div>

        <div class="col-3 align-self-center">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.user_location_show_close">
                <span class="form-check-label">关闭显示个人中心位置信息</span>
            </label>
        </div>
        <div class="col-3 align-self-center">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.user_no_ver_email_auto_delete">
                <span class="form-check-label">自动删除未验证邮箱的用户</span>
            </label>
        </div>

        <div class="col-3 align-self-center">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.user_set_avatar_gif">
                <span class="form-check-label">开启支持上传gif头像</span>
            </label>
        </div>


        <div class="col-3">
            <label class="form-label">超过几天未验证邮箱会被删除？
            </label>
            <input type="number" min="1" max="365" class="form-control" v-model="data.user_no_ver_email_auto_delete_day">
            <small>当前:{{get_options('user_no_ver_email_auto_delete_day',10)}}</small>
        </div>

        <div class="col-3">
            <label class="form-label">小黑屋用户组</label>
            <select v-model="data.user_black_group_id" class="form-select">
                <option value="0">不选择</option>
                @foreach(\App\Plugins\User\src\Models\UserClass::query()->get() as $user_group)
                    <option value="{{$user_group->id}}">{{$user_group->name}}</option>
                @endforeach
            </select>
            <small>不选则不设置小黑屋用户组</small>
        </div>

        <div class="col-12">
            <div class="card">
                <div class="card-header"><b>小黑屋用户内容重写</b></div>
                <div class="card-body row">

                    <div class="col-4">
                        <label for="" class="form-label">发布的内容</label>
                        <textarea class="form-control" v-model="data.user_ban_re_post_content" rows="3"></textarea>
                    </div>

                    <div class="col-4">
                        <label for="" class="form-label">签名</label>
                        <textarea class="form-control" v-model="data.user_ban_re_qianming" rows="3"></textarea>
                    </div>

                    <div class="col-4">
                        <label for="" class="form-label">帖子标题</label>
                        <textarea class="form-control" v-model="data.user_ban_re_topic_title" rows="3"></textarea>
                    </div>

                    <div class="col-4">
                        <label for="" class="form-label">头像链接</label>
                        <input class="form-control" v-model="data.user_ban_re_avatar" type="text" placeholder="请输入头像链接">
                    </div>

                    <div class="col-4">
                        <label for="" class="form-label">个人中心显示内容 <span class="text-red">[小部件调用代码]</span> </label>
                        <input class="form-control" v-model="data.user_ban_users_page" type="text" placeholder="输入小部件调用代码">
                    </div>

                </div>
            </div>
        </div>

    </div>

</div>