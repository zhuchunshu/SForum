@extends("app")
@section('title','邀请码管理')
@section('content')
    <div class="col-md-12" id="vue-admin-invitationCode-index">
        <div class="border-0 card">
            <div class="card-body">
                <div class="row">
                    <div class="col">
                        <h3 class="card-title">邀请码管理</h3>
                    </div>
                    <div class="col-auto">
                        <a href="/admin/Invitation-code/export" style="margin-left: 5px" class="btn btn-light">导出</a>
                        <button @@click="removeChecked" style="margin-left: 5px" v-if="show" class="btn btn-light">删除选中</button>
                    </div>
                    <div class="col-auto">
                        <span>
                            <select class="form-select" v-model="selected">
                                <option v-for="v in select_arr" :value="v.id">@{{ v.name }} </option>
                            </select>
                        </span>
                    </div>
                </div>
                <div class="table-responsive">
                    <table
                            class="table table-vcenter table-nowrap">
                        <thead>
                        <tr>
                            <th>#</th>
                            <th>邀请码</th>
                            <th>使用者</th>
                            <th>状态</th>
                            <th></th>
                            <th class="w-1"><input class="form-check-input" @change="changeAllChecked()" v-model="checked" type="checkbox"></th>
                        </tr>
                        </thead>
                        <tbody>
                            @if($page->count())
                                @foreach($page as $value)
                                    <tr>
                                        <td>{{$value->id}}</td>
                                        <td>{{$value->code}}</td>
                                        <td>@if($value->user_id) <a href="/users/{{@$value->user->username}}.html"><span class="avatar avatar-sm" style="background-image: url({{@super_avatar($value->user)}})"></span></a> @else 无 @endif</td>
                                        <td>@if($value->status) 已使用 @else 未使用 @endif</td>
                                        <td></td>
                                        <td><input v-model="checkedIds" value="{{$value->id}}" class="form-check-input" type="checkbox"></td>
                                    </tr>
                                @endforeach
                            @else
                                <tr>
                                    <td>{{__("app.none")}}</td>
                                    <td>{{__("app.none")}}</td>
                                    <td>{{__("app.none")}}</td>
                                    <td>{{__("app.none")}}</td>
                                    <td>{{__("app.none")}}</td>
                                    <td>{{__("app.none")}}</td>
                                </tr>
                            @endif
                        </tbody>
                    </table>
                </div>
            </div>
            {!! make_page($page) !!}
        </div>
    </div>
@endsection

@section('scripts')
    <script>var selected={{request()->input('where',2)}}; var page = {{request()->input('page',1)}}</script>
    <script src="{{mix('plugins/Core/js/admin.js')}}"></script>
@endsection