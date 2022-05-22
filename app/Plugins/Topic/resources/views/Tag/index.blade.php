@extends("app")

@section('title',"帖子标签管理")

@section('headerBtn')
    <a href="/admin/topic/tag/create" class="btn btn-primary">创建新标签</a>
@endsection

@section('content')
    <div class="col-md-12">
        <div class="card card-body">
            <div class="table-responsive">
                <table class="table table-vcenter table-nowrap">
                    <thead>
                    <tr>
                        <th>#</th>
                        <th>名称</th>
                        <th>图标</th>
                        <th>颜色</th>
                        <th>描述</th>
                        <th class="w-1"></th>
                        <th class="w-1"></th>
                    </tr>
                    </thead>
                    <tbody id="vue-topic-tag-table">
                    @foreach ($page as $value)
                        <tr>
                            <td>{{ $value->id }}</td>
                            <td>{{ $value->name }}</td>
                            <td>
                                <span class="avatar avatar-sm" style="background-image: url({{$value->icon}})"></span>
                            </td>
                            <td class="text-muted">
                                <div style="width: 25px;height:25px;background-color:{{ $value->color }};border-radius:5px;"></div>
                            </td>
                            @if($value->description)
                                <td class="text-muted">{{ \Hyperf\Utils\Str::limit($value->description,100) }}</td>
                            @else
                                <td class="text-muted">{{__("app.no description")}}</td>
                            @endif

                            <td>
                                <a href="/admin/topic/tag/edit/{{ $value->id }}">{{__("app.edit")}}</a>
                            </td>
                            <td>
                                <a @@click="remove({{ $value->id }})" href="#">{{__("app.delete")}}</a>
                            </td>
                        </tr>
                    @endforeach
                    </tbody>
                </table>
            </div>
            {!! make_page($page) !!}
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix('plugins/Topic/js/tag.js')}}"></script>
@endsection