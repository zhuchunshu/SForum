<div class="card card-body">
    <div class="row">
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.user.pm.msg maxlength")}}
            </label>
            <input type="number" v-model="data.pm_msg_maxlength" class="form-control">
            <small>{{__("admin.current",['current' => get_options('pm_msg_maxlength',300)])}}</small>
        </div>

        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.user.pm.msg reserve")}}
            </label>
            <input type="number" v-model="data.pm_msg_reserve" class="form-control">
            <small>{{__("admin.current",['current' => get_options('pm_msg_reserve',7)])}}({{__("admin.Unit:day")}}) |  <b class="text-red">{{__("admin.is reserved forever",['reserve' => 0])}}</b></small>
        </div>

    </div>


</div>