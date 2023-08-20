<div class="card-body">
    <div class="row">
        <div class="col-lg-4">
            <label class="form-label required">应用名称</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.APP_NAME">
            <small>当前: {{get_options('APP_NAME',env('APP_NAME','未配置'))}}</small>
        </div>

        <div class="col-lg-4">
            <label class="form-label">网站名称</label>
            <input type="text" min="1" max="3" class="form-control" v-model="data.web_name">
        </div>
        <div class="col-lg-4">
            <label class="form-label">网站标题</label>
            <input v-model="data.title" type="text" class="form-control">
        </div>
        <div class="col-lg-4">
            <label class="form-label">网站首页标题</label>
            <input v-model="data.home_title" type="text" class="form-control">
        </div>
        <div class="col-lg-4">
            <label class="form-label required">网站地址</label>
            <input v-model="data.APP_URL" type="text" class="form-control">
        </div>
{{--        <div class="col-lg-4">--}}
{{--            <label class="form-label required">网站websocket地址</label>--}}
{{--            <input v-model="data.APP_WS_URL" type="text" class="form-control">--}}
{{--        </div>--}}
        <div class="col-lg-4">
            <label class="form-label">网站关键字</label>
            <input v-model="data.keywords" type="text" class="form-control">
        </div>
        <div class="col-lg-4">
            <label class="form-label">网站描述</label>
            <input v-model="data.description" type="text" class="form-control">
        </div>
        <div class="col-lg-4">
            <label class="form-label">网站备案号</label>
            <input v-model="data.icp" type="text" class="form-control">
        </div>
        <div class="col-lg-4">
            <label class="form-label">公安备案号</label>
            <input v-model="data.ga_icp" type="text" class="form-control">
        </div>
{{--        <div class="col-lg-4">--}}
{{--            <label class="form-label">{{__('admin.setting.language')}}</label>--}}
{{--            <select v-model="data.language" class="form-select">--}}
{{--                @foreach(language()->all() as $lang=>$name)--}}
{{--                    <option value="{{$lang}}">{{$name}}</option>--}}
{{--                @endforeach--}}
{{--            </select>--}}
{{--            <small>{{__('app.default')}}: 简体中文(zh_CN)</small>--}}
{{--        </div>--}}
        <div class="col-lg-4">
            <label class="form-label">网站logo小部件调用代码</label>
            <input v-model="data.web_logo" type="text" class="form-control">
            <small>请输入小部件调用代码, <a href="/admin/hook/components" target="_blank">点我进入小部件页面</a> </small>
        </div>
        <div class="col-lg-4">
            <label class="form-label">网站自定义footer小部件调用代码</label>
            <input v-model="data.web_footer" type="text" class="form-control">
            <small>请输入小部件调用代码, <a href="/admin/hook/components" target="_blank">点我进入小部件页面</a> 为空则使用默认页脚</small>
        </div>
        <div class="col-lg-4">
            <label class="form-label">网站自定义header小部件调用代码</label>
            <input v-model="data.web_header" type="text" class="form-control">
            <small>请输入小部件调用代码, <a href="/admin/hook/components" target="_blank">点我进入小部件页面</a> 为空则使用默认页头</small>
        </div>
    </div>
</div>