<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdatePayAmountRecordTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('pay_amount_record', function (Blueprint $table) {
            $table->string('type')->default('money')->nullable();
            $table->string('change')->nullable();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('pay_amount_record', function (Blueprint $table) {
            //
        });
    }
}
