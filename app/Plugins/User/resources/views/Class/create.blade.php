@extends("app")

@section('title',"创建用户组")

@section('headerBtn')
    <a href="/admin/userClass" class="btn btn-primary">用户组管理</a>
@endsection

@section('content')
    <div class="col-md-12">
        <div class="card card-body" id="vue-user-create-class">
            <form @@submit.prevent="submit">
                <h3 class="card-title">创建用户组</h3>
                <x-csrf/>
                <div class="mb-3">
                    <label class="form-label required">名称</label>
                    <input type="text" class="form-control" v-model="name">
                </div>
                <div class="mb-3">
                    <label class="form-label required">颜色值</label>
                    <div class="row">
                        <div class="col-auto">
                            <input type="color" class="form-control form-control-color" v-model="color">

                        </div>
                        <div class="col-auto">
                            <input type="text" class="form-control"  v-model="color" >
                        </div>
                        <div class="col"></div>
                    </div>
                </div>
                <div class="mb-3">
                    <label class="form-label required">图标代码</label>
                    <textarea rows="3" v-model="icon" class="form-control"></textarea>
                    <small>填写svg代码：<a href="https://tabler-icons.io/">https://tabler-icons.io/</a> 、<a href="https://icons.bootcss.com/">https://icons.bootcss.com/</a> </small>
                </div>
                <div class="mb-3">
                    <label class="form-label required">权限(多选)</label>
                    <div class="row">
                        @foreach(Authority()->get() as $value)
                            <div class="col-4">
                                <label class="form-check form-switch">
                                    <input v-model="quanxian" value="{{$value['name']}}" class="form-check-input" type="checkbox">
                                    <span class="form-check-label">{{$value['description']}}</span>
                                </label>
                            </div>
                        @endforeach
                    </div>
                </div>
                <div class="mb-3">
                    <label class="form-label">{{__("app.permission value")}}</label>
                    <input type="number" class="form-control" v-model="permission_value">
                </div>
                <button class="btn btn-primary" type="submit">提交</button>
            </form>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{ mix("plugins/User/js/user.js") }}"></script>
@endsection

