<div class="card card-body">
    <div class="row">
        <div class="col-md-3">
            <label for="" class="form-label">
                {{__("admin.wealth.money name")}}
            </label>
            <input type="text" v-model="wealth_money_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_money_name','余额')])}}</small>
        </div>
        <div class="col-md-3">
            <label for="" class="form-label">
                {{__("admin.wealth.credit name")}}
            </label>
            <input type="text" v-model="wealth_credit_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_credit_name','积分')])}}</small>
        </div>
        <div class="col-md-3">
            <label for="" class="form-label">
                {{__("admin.wealth.golds name")}}
            </label>
            <input type="text" v-model="wealth_golds_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_golds_name','金币')])}}</small>
        </div>
        <div class="col-md-3">
            <label for="" class="form-label">
                {{__("admin.wealth.exp name")}}
            </label>
            <input type="text" v-model="wealth_exp_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_exp_name','经验')])}}</small>
        </div>
    </div>


</div>