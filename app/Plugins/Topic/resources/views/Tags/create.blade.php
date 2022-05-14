@extends("App::app")
@section('title','创建标签')

@section('content')
    <div class="col-md-12">
        <div class="card card-body" id="vue-user-topic-tag-create">
            <h3 class="card-title">新增标签</h3>
            <form method="post" action="/tags/create?Redirect=/tags/create" @@submit.prevent="submit" enctype="multipart/form-data">
                <x-csrf/>
                <div class="mb-3">
                    <label class="form-label">
                        名称
                    </label>
                    <input type="text" class="form-control" name="name" v-model="name" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        颜色
                    </label>
                    <input type="color" class="form-control form-control-color" name="color" v-model="color" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        图标
                    </label>
                    <input type="file" accept="image/gif, image/png, image/jpeg, image/jpg" class="form-control" name="icon" v-model="icon" required>
                </div>
                <div class="mb-3">
                    <label class="form-label"> 哪个用户组可使用此标签? </label>
                    <select class="form-select" name="userClass[]" multiple size="8">
                        @foreach($userClass as $value)
                            <option value="{{$value->name}}">{{$value->name}}</option>
                        @endforeach
                    </select>
                    <small style="color:red">不选择则所有用户组都可用此标签</small>
                </div>
                <div class="mb-3">
                    <label class="form-label">描述</label>
                    <textarea name="description" class="form-control" rows="4"></textarea>
                </div>
                <div class="mb-3">
                    <button class="btn btn-primary">提交</button>
                </div>
            </form>
        </div>
    </div>
@endsection

