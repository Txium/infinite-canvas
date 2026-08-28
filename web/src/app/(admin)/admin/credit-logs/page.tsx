"use client";

import { ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { ProTable, type ProColumns } from "@ant-design/pro-components";
import { Button, Card, Col, Form, Input, Row, Space, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";

import type { AdminCreditLog } from "@/services/api/admin";
import { formatCNY } from "@/constant/credits";
import { useAdminCreditLogs } from "./use-admin-credit-logs";

const creditLogTypeLabels: Record<string, string> = {
    admin_adjust: "后台调整",
    ai_consume: "模型消费",
    ai_refund: "失败返还",
    ai_freeze: "生成冻结",
    ai_settle: "成功结算",
    ai_release: "失败解冻",
    recharge: "充值到账",
};

export default function AdminCreditLogsPage() {
    const { logs, keyword, page, pageSize, total, isLoading, searchLogs, changePage, changePageSize, resetFilters, refreshLogs } = useAdminCreditLogs();
    const [keywordText, setKeywordText] = useState(keyword);

    useEffect(() => setKeywordText(keyword), [keyword]);

    const columns: ProColumns<AdminCreditLog>[] = [
        {
            title: "用户 ID",
            dataIndex: "userId",
            width: 220,
            render: (_, item) => <Typography.Text copyable>{item.userId}</Typography.Text>,
        },
        {
            title: "类型",
            dataIndex: "type",
            width: 140,
            render: (_, item) => <Tag>{creditLogTypeLabels[item.type] || item.type || "-"}</Tag>,
        },
        {
            title: "变动",
            dataIndex: "amount",
            width: 100,
            render: (_, item) => <Typography.Text type={item.amount >= 0 ? "success" : "danger"}>{item.amount >= 0 ? "+" : "-"}{formatCNY(Math.abs(item.amount))}</Typography.Text>,
        },
        {
            title: "可用余额",
            dataIndex: "balance",
            width: 110,
            render: (_, item) => formatCNY(item.balance),
        },
        {
            title: "冻结变动",
            dataIndex: "frozenAmount",
            width: 110,
            render: (_, item) => item.frozenAmount === 0 ? "-" : `${item.frozenAmount > 0 ? "+" : "-"}${formatCNY(Math.abs(item.frozenAmount))}`,
        },
        {
            title: "冻结余额",
            dataIndex: "frozenBalance",
            width: 110,
            render: (_, item) => formatCNY(item.frozenBalance),
        },
        {
            title: "备注",
            dataIndex: "remark",
            ellipsis: true,
            render: (_, item) => <Typography.Text type="secondary">{item.remark || "-"}</Typography.Text>,
        },
        {
            title: "创建时间",
            dataIndex: "createdAt",
            width: 180,
            render: (_, item) => <Typography.Text type="secondary">{item.createdAt ? dayjs(item.createdAt).format("YYYY-MM-DD HH:mm:ss") : "-"}</Typography.Text>,
        },
    ];

    return (
        <main style={{ padding: 24 }}>
            <Space direction="vertical" size={16} style={{ width: "100%" }}>
                <Card variant="borderless">
                    <Form layout="vertical">
                        <Row gutter={16} align="bottom">
                            <Col flex="360px">
                                <Form.Item label="关键词">
                                    <Input.Search value={keywordText} placeholder="搜索用户 ID、类型、备注或关联 ID" allowClear enterButton={<SearchOutlined />} onSearch={() => searchLogs(keywordText)} onChange={(event) => setKeywordText(event.target.value)} />
                                </Form.Item>
                            </Col>
                            <Col flex="none">
                                <Form.Item>
                                    <Space>
                                        <Button
                                            onClick={() => {
                                                setKeywordText("");
                                                resetFilters();
                                            }}
                                        >
                                            重置
                                        </Button>
                                        <Button type="primary" icon={<ReloadOutlined />} onClick={() => searchLogs(keywordText)}>
                                            查询
                                        </Button>
                                    </Space>
                                </Form.Item>
                            </Col>
                        </Row>
                    </Form>
                </Card>
                <ProTable<AdminCreditLog>
                    rowKey="id"
                    columns={columns}
                    dataSource={logs}
                    loading={isLoading}
                    search={false}
                    defaultSize="middle"
                    tableLayout="fixed"
                    cardProps={{ variant: "borderless" }}
                    headerTitle={
                        <Space>
                            <Typography.Text strong>钱包流水</Typography.Text>
                            <Tag>{total} 条</Tag>
                        </Space>
                    }
                    options={{ density: true, setting: true, reload: () => void refreshLogs() }}
                    pagination={{
                        current: page,
                        pageSize,
                        total,
                        showSizeChanger: true,
                        pageSizeOptions: [10, 20, 50, 100],
                        showTotal: (value) => `共 ${value} 条`,
                        onChange: (nextPage, nextPageSize) => (nextPageSize !== pageSize ? changePageSize(nextPageSize) : changePage(nextPage)),
                    }}
                />
            </Space>

        </main>
    );
}
