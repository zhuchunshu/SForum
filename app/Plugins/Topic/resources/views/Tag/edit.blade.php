@extends("app")

@section('title',"修改id为".$data->id."的帖子标签")

@section('content')
    <div class="col-md-12">
        <div class="card card-body" id="vue-topic-tag-edit">
            <h3 class="card-title">修改id为{{$data->id}}的帖子标签</h3>
            <form method="post" action="/admin/topic/tag/edit?Redirect=/admin/topic/tag/edit/{{$data->id}}" @@submit.prevent="submit" enctype="multipart/form-data">
                <x-csrf/>
                <div class="mb-3">
                    <label class="form-label">
                        名称
                    </label>
                    <input type="text" class="form-control" value="{{$data->name}}" name="name" v-model="name" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        颜色
                    </label>
                    <input type="color" class="form-control form-control-color" value="{{$data->color}}" name="color" v-model="color" required>
                </div>
                <div class="mb-3">
                    <label class="form-label">
                        图标
                        <span class="avatar" style="background-image: url({{$data->icon}})"></span>
                    </label>
                    <input type="file" accept="image/gif, image/png, image/jpeg, image/jpg" class="form-control" name="icon" v-model="icon">
                </div>
                <div class="mb-3">
                    <label class="form-label">描述</label>
                    <textarea name="description" class="form-control" rows="4">{{$data->description}}</textarea>
                </div>
                <div class="mb-3">
                    <button name="id" value="{{$data->id}}" class="btn btn-primary">提交</button>
                </div>
            </form>
        </div>
    </div>
@endsection


@section("scripts")
    <script src="{{mix('plugins/Topic/js/tag.js')}}"></script>
@endsection
