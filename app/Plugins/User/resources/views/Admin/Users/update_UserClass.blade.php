@extends("app")

@section('title',"修改".$data->username."的用户组")


@section('content')
    <div class="col-md-12">
        <div class="row row-cards">
            <div class="col-md-4">
                <div class="border-0 card">
                    <div class="card-body">
                        <h3 class="card-title">修改用户组</h3>
                        <form action="/admin/users/update/UserClass" method="post">
                            <x-csrf/>
                            <input type="hidden" name="user_id" value="{{$data->id}}">
                            <div class="mb-3">
                                <label class="form-label">
                                    选择用户组
                                </label>
                                <select name="class_id" class="form-select">
                                    @foreach($class as $value)
                                        <option value="{{$value->id}}" @if ($value->id===$data->Class->id) selected @endif>{{$value->name}}</option>
                                    @endforeach
                                </select>
                            </div>
                            <button type="submit" class="btn btn-primary">提交</button>
                        </form>
                    </div>
                </div>
            </div>

            <div class="col-md-4">
                <div class="border-0 card">
                    <div class="card-body">
                        <h3 class="card-title">用户组信息</h3>
                        <div class="accordion" id="accordion-example">
                            @foreach($class as $value)
                                <div class="accordion-item">
                                    <h2 class="accordion-header" id="heading-{{$value->id}}">
                                        <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#collapse-{{$value->id}}" aria-expanded="true">
                                            <div style="width: 25px;height:25px;background-color:{{$value->color}};border-radius:5px;margin-right: 5px"></div>{{$value->name}}
                                        </button>
                                    </h2>
                                    <div id="collapse-{{$value->id}}" class="accordion-collapse collapse" data-bs-parent="#accordion-example">
                                        <div class="accordion-body pt-0">
                                            <b>{{__("app.permission value")}}:</b> {{$value['permission-value']}}
                                            <div>
                                                <h3>权限列表</h3>
                                                @foreach(json_decode($value->quanxian) as $quanxian)
                                                    <span class="badge bg-blue">{{Authority()->getName($quanxian)}}</span>
                                                @endforeach
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            @endforeach
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection
